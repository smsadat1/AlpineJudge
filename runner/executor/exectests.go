package executor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"shared"
	"strings"
	"syscall"
	"time"
	"utils"

	containerd "github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
)

func ExecSubm(
	ctx context.Context,
	container containerd.Container,
	rules utils.ExecRules,
	jobspec shared.JobSpec,
	rmqm shared.RMQManager,
	s3m shared.S3Manager,
) utils.ContainerInfo {

	contInfo := utils.ContainerInfo{
		SubmissionId:    jobspec.SubmissionID,
		Language:        jobspec.Language,
		Version:         jobspec.Version,
		Interval:        0,
		Status:          "Pending",
		StatusInfo:      "",
		ContainerStdout: "",
		ContainerStderr: "",
	}

	_ = "submissions/" + jobspec.SubmissionID + "/"
	var stdoutWriter bytes.Buffer
	var stderrWrite bytes.Buffer

	start := time.Now()

	// Setup unix socket
	listener, err := net.Listen("unix", rules.HostEventSocket)
	if err != nil {
		log.Fatalf("Failed to create socket listener: %v", err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.RemoveAll(rules.HostEventSocket)
	}()

	// Context to gracefully shut down socket listener worker when function returns
	sockCtx, sockCancel := context.WithCancel(ctx)
	defer sockCancel()

	// Goroutine: Accepts incoming socket connections from container & publishes to RabbitMQ in real time
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-sockCtx.Done():
					return // Normal exit when task finishes
				default:
					log.Printf("Socket accept error: %v", err)
					return
				}
			}

			// Handle stream from this connection
			go func(c net.Conn) {
				defer c.Close()

				scanner := bufio.NewScanner(c)
				if err := scanner.Err(); err != nil {
					log.Printf("Failed to scan streamed data: %v", err)
					return
				}
				streamStart := time.Now()
				for scanner.Scan() {
					eventPayload := scanner.Bytes()

					// pass Type, Status & Details in RMQ
					routeToRMQ(sockCtx, rules.EventQueueName, rmqm, eventPayload)

					// Process S3 uploads in a goroutine so it doesn't block scanner.Scan()
					payloadCopy := make([]byte, len(eventPayload))
					go func(data []byte) {
						var forS3 utils.Event
						if err := json.Unmarshal(data, &forS3); err == nil {
							s3m.UploadFileToS3(sockCtx, "submissions/"+jobspec.SubmissionID+"/stdout.log", strings.NewReader(forS3.Stdout))
							s3m.UploadFileToS3(sockCtx, "submissions/"+jobspec.SubmissionID+"/stderr.log", strings.NewReader(forS3.Stderr))
						}
					}(payloadCopy)
				}
				streamInterval := time.Since(streamStart).Milliseconds()
				fmt.Printf("\nSubmission execution interval: %vms\n", streamInterval)

			}(conn)
		}
	}()

	task, err := container.NewTask(
		ctx,
		cio.NewCreator(cio.WithStreams(nil, &stdoutWriter, &stderrWrite)),
	)

	if err != nil {

		log.Printf("NewTask RPC error: %v", err)

		contInfo.Interval = 0
		contInfo.Status = utils.VerdictIE
		contInfo.StatusInfo = "Failed to create container task"
		return contInfo
	}

	defer task.Delete(ctx)

	// obtain wait channel before task.Start()
	statusCode, err := task.Wait(ctx)
	if err != nil {

		log.Printf("NewTask RPC error: %v", err)

		contInfo.Interval = 0
		contInfo.Status = utils.VerdictIE
		contInfo.StatusInfo = "Failed to obtain wait status channel"
		return contInfo
	}

	if err := task.Start(ctx); err != nil {

		log.Printf("NewTask RPC error: %v", err)

		contInfo.Interval = 0
		contInfo.Status = utils.VerdictIE
		contInfo.StatusInfo = "Failed to start container task"
		return contInfo
	}

	// Handle timeouts & exit
	timeoutDuration := time.Duration(rules.Timeoutsec)*time.Second + 5 // extra 5 seconds at container level
	ctxTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	var status containerd.ExitStatus
	select {
	case status = <-statusCode:

		// Process completed naturally within timeout limit
		elapsedMS := time.Since(start).Milliseconds()
		contInfo.Interval = uint64(elapsedMS)

		if status.ExitCode() == 0 {
			contInfo.Status = utils.VerdictAC
			contInfo.StatusInfo = "Container exited normally"
			contInfo.ContainerStderr = stderrWrite.String()
			contInfo.ContainerStdout = stdoutWriter.String()
		} else {
			contInfo.Status = utils.VerdictRE
			contInfo.StatusInfo = fmt.Sprintf("Container exited with code %d", status.ExitCode())
			contInfo.ContainerStderr = stderrWrite.String()
			contInfo.ContainerStdout = stdoutWriter.String()
		}
	case <-ctxTimeout.Done():
		// TLE
		elapsedMS := time.Since(start).Milliseconds()
		contInfo.Interval = uint64(elapsedMS)
		contInfo.Status = utils.VerdictTLE
		contInfo.StatusInfo = fmt.Sprintf("Task exceeded time limit of %d seconds", rules.Timeoutsec)
		contInfo.ContainerStderr = stderrWrite.String()
		contInfo.ContainerStdout = stdoutWriter.String()

		log.Print("Task timedout. Sending SIGKILL to container...")
		_ = task.Kill(ctx, syscall.SIGKILL)
	}

	return contInfo
}
