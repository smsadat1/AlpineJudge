package pkg

import (
	"context"
	"fmt"
	"local/runner/internal"
	"log"
	"math/rand/v2"
	"os"
	"time"

	containerd "github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	amqp "github.com/rabbitmq/amqp091-go"

	"utils"
)

func InitRunner(ctx context.Context, deps utils.EngineDeps) error {

	// Subdirectories needed for the rbind mount to containers
	dirs := []string{
		"/tmp/runner/sockets",
		"/tmp/runner/testsets",
		"/tmp/runner/submissions",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	log.Println("Base directories initialized at /tmp/runner")

	localQueue := make(chan amqp.Delivery, 100)
	containerQueue := make(chan *internal.WarmContainer, 15)

	log.Println("Creating container namespace...")
	cCtx := namespaces.WithNamespace(ctx, "alpine_judge")

	// RMQ consumer loop
	if err := deps.Rmq.Subscribe(ctx, localQueue, deps.JobQueue, "rmq-consoomer"); err != nil {
		return fmt.Errorf("RMQ consumer failed: %v\n", err)
	}

	// Warm Container Producer Loop
	go func() {
		countainerCounter := 0
		pcgsrc := rand.NewPCG(1, 10000)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				countainerCounter++
				fmt.Printf("Creating container %d\n", countainerCounter)

				slotID := rand.New(pcgsrc)

				warmContainer, err := internal.CreateWarmContainer(cCtx, deps.Client, slotID.Uint32())
				if err != nil {
					log.Printf("Warm Pool Error: failed to create container: %v", err)
					time.Sleep(1 * time.Second)
					continue
				}

				select {
				case containerQueue <- warmContainer:
				case <-ctx.Done():
					_ = warmContainer.Container.Delete(cCtx, containerd.WithSnapshotCleanup)
					return
				}
			}
		}
	}()

	log.Println("Runner Daemon successfully initialized and monitoring...")

	// Main orchestration loop
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down Runner daemon loops gracefully...")
			log.Println("Cleaning up artifacts...")
			if err := os.RemoveAll("/tmp/runner"); err != nil {
				log.Printf("Warning: artifact cleanup failed: %v", err)
			}
			return nil

		case msg := <-localQueue:

			go func(delivery amqp.Delivery) {

				var warmCont *internal.WarmContainer
				select {
				case warmCont = <-containerQueue:
				case <-time.After(3 * time.Second):
					log.Println("Warm pool starved, falling back to on-demand creation...")

					pcgsrc := rand.NewPCG(1, 10000)
					slotID := rand.New(pcgsrc)

					var err error
					warmCont, err = internal.CreateWarmContainer(cCtx, deps.Client, slotID.Uint32())
					if err != nil {
						log.Printf("Fallback container creation failed: %v", err)
						_ = delivery.Nack(false, true)
						return
					}
				}

				jobspec, err := utils.ProcessJobSpec(ctx, delivery)
				if err != nil {
					log.Printf("Jobspec Parse Error: %v", err)
					_ = delivery.Nack(false, true)
					return
				}

				// Execute using warm container or fallback to on-demand
				contInfo, err := internal.OrchestrateSubm(ctx, cCtx, warmCont, *deps.S3, jobspec, *deps.Rmq)
				log.Printf("Container Info -> SubmissionID: %v | Status: %v | Info: %v\n Container stdout: %v | Container stderr: %v",
					contInfo.SubmissionId, contInfo.Status, contInfo.StatusInfo, contInfo.ContainerStdout, contInfo.ContainerStderr)
				if err != nil {
					log.Printf("Execution Failure: %v", err)
					_ = delivery.Nack(false, false)
					return
				}

				_ = delivery.Ack(false) // notify RMQ that task is cleared
				log.Printf("Task successfully executed. Slot released.")
			}(msg)
		}
	}
}
