// manages running container
package executor

import (
	"context"
	"local/runner/utils"
	"log"
	"syscall"
	"time"

	"github.com/containerd/containerd/errdefs"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
)

func manageContainer(
	ctx context.Context, container containerd.Container, rules utils.ExecRules,
) (utils.ResultSpec, error) {

	// pass the write end to containerd
	task, err := container.NewTask(ctx, cio.NewCreator())

	if err != nil {
		return utils.ResultSpec{}, err
	}
	// made sure to delete if something fails midway
	defer task.Delete(ctx)

	// get exit status channel
	statusC, err := task.Wait(ctx)
	if err != nil {
		return utils.ResultSpec{}, err
	}

	// start task execution
	if err := task.Start(ctx); err != nil {
		return utils.ResultSpec{}, err
	}
	log.Println("Task started successfully")

	timeoutDuration := time.Duration(rules.Timeoutsec) * time.Second
	ctxTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	// dynamic wait block
	select {
	case status := <-statusC:

		log.Println("Task completed.")
		if status.Error() != nil {
			return utils.ResultSpec{}, status.Error()
		}

	case <-ctxTimeout.Done():
		// force kill , just in case
		log.Printf("[nsrunner] Task exceeded set timeout %v\nStopping task...\n", rules.Timeoutsec)
		if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
			if errdefs.IsNotFound(err) {
				log.Println("Task finished right as timeout hit; ignoring 'not found' error.")
			} else {
				// genuine error
				return utils.ResultSpec{}, err
			}
		}
	}

	// block till exit status
	status := <-statusC
	if status.Error() != nil {
		return utils.ResultSpec{}, status.Error()
	}

	log.Printf("[nsrunner] Task exited with status code %v", status.ExitCode())

	return utils.ResultSpec{}, nil
}
