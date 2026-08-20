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
		Interval:        0,
		Status:          "PENDING",
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
		maxlogcapKB, exists := os.LookupEnv("MAX_LOG_CAP_KB")

		if !exists {
			log.Fatal("Missing env var DIRECT_EXCHANGE_NAME")
		}
		scanner := bufio.NewScanner(c)
		buf := make([]byte, 60*1024) // 60 KB buffer size
		maxlogcapKBn, err := strconv.ParseInt(maxlogcapKB, 10, 32)
		if err != nil {
			log.Fatalf("Failed converting maxlogcapKB to int: %v", err)
		}
		scanner.Buffer(buf, int(maxlogcapKBn*1024)) // mutliplied with 1024 to make KB size

		exchangename, exists := os.LookupEnv("DIRECT_EXCHANGE_NAME")
		if !exists {
			log.Fatal("Missing env var DIRECT_EXCHANGE_NAME")
		}

		for scanner.Scan() {
			eventPayload := scanner.Bytes()

			var eventStream utils.Event
			json.Unmarshal(eventPayload, &eventStream)

			rmqPayload := utils.RMQPayload{
				Type:    eventStream.Type,
				Status:  eventStream.Status,
				Details: eventStream.Details,
			}
			rmqData, err := json.Marshal(&rmqPayload)
			if err != nil {
				log.Printf("Error marshaling json data to rmqdata: %v\n", err)
			}

			routeToRMQ(ctx, jobspec.SubmissionID, rmqm, exchangename, rmqData)

			// S3 upload happens asynchronously so uplaod doesn't block main event stream
			go func() {
				if err := s3m.UploadFileToS3(
					ctx, fmt.Sprintf("%v/result/stdout.log", jobspec.SubmissionID), strings.NewReader(eventStream.Stdout),
				); err != nil {
					// in case of error, log it and move on. Can't wait during live stream
					log.Printf("Error uploading stdout.log to S3: %v\n", err)
				}
			}()

			go func() {
				if err := s3m.UploadFileToS3(
					ctx, fmt.Sprintf("%v/result/stderr.log", jobspec.SubmissionID), strings.NewReader(eventStream.Stderr),
				); err != nil {
					// in case of error, log it and move on. Can't wait during live stream
					log.Printf("Error uploading stderr.log to S3: %v\n", err)
				}
			}()
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Failed to scan streamed data: %v", err)
			return
		}

	}(wc.Conn)

	// Handle timeouts & exit
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
			contInfo.Status = "OK"
			contInfo.StatusInfo = "Container exited normally"
			contInfo.ContainerStderr = stderrWrite.String()
			contInfo.ContainerStdout = stdoutWriter.String()
		} else {
			contInfo.Status = "ERROR"
			contInfo.StatusInfo = fmt.Sprintf("Container exited with code %d", status.ExitCode())
			contInfo.ContainerStderr = stderrWrite.String()
			contInfo.ContainerStdout = stdoutWriter.String()
		}
	case <-ctxTimeout.Done():
		// TLE
		elapsedMS := time.Since(start).Milliseconds()
		contInfo.Interval = uint64(elapsedMS)
		contInfo.Status = "ERROR"
		contInfo.StatusInfo = fmt.Sprintf("Task exceeded time limit of %d seconds", timeoutDuration)
		contInfo.ContainerStderr = stderrWrite.String()
		contInfo.ContainerStdout = stdoutWriter.String()

		log.Print("Task timedout. Sending SIGKILL to container...")
		_ = wc.Task.Kill(ctx, syscall.SIGKILL)
	}

	return contInfo
}
