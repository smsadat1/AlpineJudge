package internal

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
	"strconv"
	"strings"
	"syscall"
	"time"
	"utils"

	containerd "github.com/containerd/containerd"
	"github.com/joho/godotenv"
)

func (wc *WarmContainer) ExecSubm(
	ctx context.Context,
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

	start := time.Now()

	// Context to gracefully shut down socket listener worker when function returns
	// sockCtx, sockCancel := context.WithCancel(ctx)
	// defer sockCancel()

	// Goroutine: Accepts incoming socket connections from container & publishes to RabbitMQ in real time

	// Handle stream from this connection
	go func(c net.Conn) {
		defer c.Close()

		scanner := bufio.NewScanner(c)
		if err := scanner.Err(); err != nil {
			log.Printf("Failed to scan streamed data: %v", err)
			return
		}

		for scanner.Scan() {
			eventPayload := scanner.Bytes()

			var eventdata utils.Event
			json.Unmarshal(eventPayload, &eventdata)
			fmt.Printf("\nEvent: %v\n", eventdata)

			// pass Type, Status & Details in RMQ
			routeToRMQ(ctx, jobspec.SSEQueue, rmqm, eventPayload)

			// Process S3 uploads in a goroutine so it doesn't block scanner.Scan()
			payloadCopy := make([]byte, len(eventPayload))
			go func(data []byte) {
				var forS3 utils.Event
				if err := json.Unmarshal(data, &forS3); err == nil {
					s3m.UploadFileToS3(ctx, "submissions/"+jobspec.SubmissionID+"/stdout.log", strings.NewReader(forS3.Stdout))
					s3m.UploadFileToS3(ctx, "submissions/"+jobspec.SubmissionID+"/stderr.log", strings.NewReader(forS3.Stderr))
				}
			}(payloadCopy)
		}

	}(wc.Conn)

	// Handle timeouts & exit

	godotenv.Load(".env")
	timeouts, _ := strconv.ParseInt(os.Getenv("TIMEOUT_SEC"), 10, 64)
	timeoutDuration := time.Duration(timeouts) * time.Second
	ctxTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	var status containerd.ExitStatus
	var stdoutWriter bytes.Buffer
	var stderrWrite bytes.Buffer
	select {
	case status = <-wc.ContStatus:

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
		contInfo.StatusInfo = fmt.Sprintf("Task exceeded time limit of %d seconds", timeoutDuration)
		contInfo.ContainerStderr = stderrWrite.String()
		contInfo.ContainerStdout = stdoutWriter.String()

		log.Print("Task timedout. Sending SIGKILL to container...")
		_ = wc.Task.Kill(ctx, syscall.SIGKILL)
	}

	return contInfo
}
