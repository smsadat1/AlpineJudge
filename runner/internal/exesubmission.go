package internal

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"shared"
	"strconv"
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

	// Goroutine: Accepts incoming socket connections from container & publishes to RabbitMQ in real time
	// Handle stream from this connection
	go func(c net.Conn) {
		defer c.Close()

		/*
			By default unix socket only have 64KB of log limit, which is ofter far below than a testcase might send during OLE
			Which causes overflow and unix socket connection drops silently.
			That's why a good estimation of 10 MB max log size is set with 60KB of buffer so socket connection doesn't break during OLE
		*/
		// TODO: Make MAXLOGCAP part of env vars instead
		const MAXLOGCAP = 10 * 1024 * 1024 // 10 MB
		scanner := bufio.NewScanner(c)
		buf := make([]byte, 60*1024)
		scanner.Buffer(buf, MAXLOGCAP)

		for scanner.Scan() {
			eventPayload := scanner.Bytes()

			// var eventdata utils.Event
			// json.Unmarshal(eventPayload, &eventdata)
			// fmt.Printf("\nEvent: %v\n", eventdata)

			// pass entire payload to RMQ
			// Earlier only Type, Status & Detail was sent while passing stdout & stderr directed to S3
			routeToRMQ(ctx, jobspec.SSEQueue, rmqm, eventPayload)
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Failed to scan streamed data: %v", err)
			return
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
