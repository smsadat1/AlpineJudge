package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"shared"
	"syscall"
	"time"
	"utils"

	containerd "github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	amqp "github.com/rabbitmq/amqp091-go"
)

func ExecSubm(
	ctx context.Context,
	container containerd.Container,
	rules utils.ExecRules,
	jobspec shared.JobSpec,
	rmqm shared.RMQManager,
	s3m shared.S3Manager,
) utils.ResultSpec {

	result := utils.ResultSpec{
		SubmissionId: jobspec.SubmissionID,
		Language:     jobspec.Language,
		Version:      jobspec.Version,
		Interval:     0,
		Status:       "Pending",
		Details:      "",
	}

	s3keyPrefix := "submissions/" + jobspec.SubmissionID + "/"
	var stdoutWriter bytes.Buffer
	var stderrWrite bytes.Buffer

	// Setup unix socket
	_ = os.Remove(rules.HostEventSocket) // cleanup stale socket
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
				for scanner.Scan() {
					eventPayload := scanner.Bytes()

					// publish event payload directly to RMQ in real time
					msg := amqp.Publishing{
						ContentType:  "application/json",
						DeliveryMode: amqp.Persistent,
						Timestamp:    time.Now(),
						Body:         eventPayload,
					}

					if err := rmqm.Publish(sockCtx, rules.EventQueueName, msg); err != nil {
						log.Printf("Failed to stream event to RMQ: %v", err)
					}
				}

			}(conn)
		}
	}()

	// start container task
	task, err := container.NewTask(
		ctx,
		cio.NewCreator(cio.WithStreams(nil, &stdoutWriter, &stderrWrite)),
	)

	if err != nil {

		log.Printf("NewTask RPC error: %v", err)

		s3m.UploadFileToS3(ctx, s3keyPrefix+"stdout.log", bytes.NewReader(stdoutWriter.Bytes()))
		s3m.UploadFileToS3(ctx, s3keyPrefix+"stderr.log", bytes.NewReader(stderrWrite.Bytes()))

		result.Interval = 0
		result.Status = utils.VerdictIE
		result.Details = "Failed to create container task"
		return result
	}

	defer task.Delete(ctx)

	// obtain wait channel before task.Start()
	statusCode, err := task.Wait(ctx)
	if err != nil {

		log.Printf("NewTask RPC error: %v", err)

		s3m.UploadFileToS3(ctx, s3keyPrefix+"stdout.log", bytes.NewReader(stdoutWriter.Bytes()))
		s3m.UploadFileToS3(ctx, s3keyPrefix+"stderr.log", bytes.NewReader(stderrWrite.Bytes()))

		result.Interval = 0
		result.Status = utils.VerdictIE
		result.Details = "Failed to obtain wait status channel"
		return result
	}

	start := time.Now()

	if err := task.Start(ctx); err != nil {

		log.Printf("NewTask RPC error: %v", err)

		s3m.UploadFileToS3(ctx, s3keyPrefix+"stdout.log", bytes.NewReader(stdoutWriter.Bytes()))
		s3m.UploadFileToS3(ctx, s3keyPrefix+"stderr.log", bytes.NewReader(stderrWrite.Bytes()))

		result.Interval = 0
		result.Status = utils.VerdictIE
		result.Details = "Failed to start container task"
		return result
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
		result.Interval = uint64(elapsedMS)

		if status.Error() != nil {
			result.Status = utils.VerdictIE
			result.Details = "Task exited abnormally"
		} else if status.ExitCode() != 0 {
			result.Status = utils.VerdictIE
			result.Details = fmt.Sprintf("Task exited with non-zero exit code: %d", status.ExitCode())
		} else {
			result.Status = utils.VerdictAC
			result.Details = "Task executed successfully"
		}

	case <-ctxTimeout.Done():
		// TLE
		elapsedMS := time.Since(start).Milliseconds()
		result.Interval = uint64(elapsedMS)
		result.Status = utils.VerdictTLE
		result.Details = fmt.Sprintf("Task exceeded time limit of %d seconds", rules.Timeoutsec)

		log.Print("Task timedout. Sending SIGKILL to container...")
		_ = task.Kill(ctx, syscall.SIGKILL)
	}

	// TODO: Fix stdout & stderr being empty issue
	log.Printf(" Stdout output: %s", stdoutWriter.String())
	log.Printf(" Stderr output: %s", stderrWrite.String())

	s3m.UploadFileToS3(ctx, s3keyPrefix+"stdout.log", bytes.NewReader(stdoutWriter.Bytes()))
	s3m.UploadFileToS3(ctx, s3keyPrefix+"stderr.log", bytes.NewReader(stderrWrite.Bytes()))

	return result
}
