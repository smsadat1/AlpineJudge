package executor

import (
	"context"
	"io"
	"log"
	"os"
	"shared"
	"syscall"
	"time"

	"github.com/containerd/containerd/errdefs"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/rabbitmq/amqp091-go"

	"local/runner/utils"
)

func execSubm(
	ctx context.Context, container containerd.Container, rules utils.ExecRules, jobspec utils.JobSpec, rmqm shared.RMQManager,
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
	localQueue := make(chan amqp091.Publishing, 100)

	task, err := container.NewTask(
		ctx,
		cio.NewCreator(cio.WithStreams(os.Stdin, stdoutWriter, stderrWriter)),
	)
	if err != nil {
		result.Status = utils.VerdictIE
		return result
	}

	defer task.Delete(ctx)

	// get real time log stream
	go utils.StreamContainerLogsToRMQ(ctx, rules.OutStreamQueueName, stdoutReader, rmqm, localQueue)
	go utils.StreamContainerLogsToRMQ(ctx, rules.ErrStreamQueueName, stderrReader, rmqm, localQueue)

	// get exit status channel
	statusC, err := task.Wait(ctx)
	if err != nil {
		result.Status = utils.VerdictCE
		return result
	}

	// start task execution
	if err := task.Start(ctx); err != nil {
		result.Status = utils.VerdictIE
		return result
	}

	timeoutDuration := time.Duration(rules.Timeoutsec) * time.Second
	ctxTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	// dynamic wait block
	select {
	case status := <-statusC:

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

	// block till exit status
	status := <-statusC
	if status.Error() != nil {
		result.Status = utils.VerdictWA
		return result
	}

	log.Printf("Container task exited with status code %v", status.ExitCode())

	result.Status = utils.VerdictAC
	return result
}
