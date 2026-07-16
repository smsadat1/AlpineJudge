package executor

import (
	"context"
	"io"
	"log"
	"shared"
	"sync"
	"syscall"
	"time"

	containerd "github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/errdefs"
	amqp "github.com/rabbitmq/amqp091-go"

	"local/runner/utils"
)

func execSubm(
	ctx context.Context, container containerd.Container, rules utils.ExecRules, jobspec shared.JobSpec, rmqm shared.RMQManager,
) utils.ResultSpec {

	result := utils.ResultSpec{
		SubmissionId: jobspec.SubmissionID,
		Language:     jobspec.Language,
		Version:      jobspec.Version,
		Interval:     "0",
		Status:       "Pending",
	}
	// synchronous unix pipe for read & write
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	localQueue := make(chan amqp.Publishing, 100)

	task, err := container.NewTask(
		ctx,
		cio.NewCreator(cio.WithStreams(nil, stdoutWriter, stderrWriter)),
	)
	if err != nil {
		stderrWriter.Close()
		stdoutWriter.Close()
		result.Status = utils.VerdictIE
		return result
	}

	defer task.Delete(ctx)

	// get exit status channel
	statusC, err := task.Wait(ctx)
	if err != nil {
		stderrWriter.Close()
		stdoutWriter.Close()
		result.Status = utils.VerdictCE
		return result
	}

	// start task execution
	if err := task.Start(ctx); err != nil {
		stderrWriter.Close()
		stdoutWriter.Close()
		result.Status = utils.VerdictIE
		return result
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case msg, ok := <-localQueue:
			if ok {
				utils.StreamContainerLogsToRMQ(ctx, rules.OutStreamQueueName, stdoutReader, rmqm, msg)
			}
		case <-ctx.Done():
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case msg, ok := <-localQueue:
			if ok {
				utils.StreamContainerLogsToRMQ(ctx, rules.ErrStreamQueueName, stderrReader, rmqm, msg)
			}
		case <-ctx.Done():
		}
	}()

	timeoutDuration := time.Duration(rules.Timeoutsec)*time.Second + 5 // extra 5 seconds at container level
	ctxTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	// dynamic wait block
	var status containerd.ExitStatus
	select {
	case status = <-statusC:

		log.Println("Task completed.")
		if status.Error() != nil {
			result.Status = utils.VerdictIE
			return result
		}

	case <-ctxTimeout.Done():
		// force kill , just in case
		log.Printf("Task exceeded set timeout %v\nStopping task...\n", rules.Timeoutsec)
		if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
			if errdefs.IsNotFound(err) {
				log.Println("Task finished right as timeout hit; ignoring 'not found' error.")
			} else {
				// genuine error
				result.Status = utils.VerdictTLE
				return result
			}
		}
	}

	stderrWriter.Close()
	stdoutWriter.Close()

	// wait for  logging goroutines to fully finish scanning and flush remaining logs
	wg.Wait()

	// block till exit status
	if status.ExitCode() != 0 {
		result.Status = utils.VerdictWA
		return result
	}

	log.Printf("Container task exited with status code %v", status.ExitCode())
	result.Status = utils.VerdictAC
	return result
}
