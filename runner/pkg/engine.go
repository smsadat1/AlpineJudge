package pkg

import (
	"context"
	"fmt"
	"local/runner/internal"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"utils"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	amqp "github.com/rabbitmq/amqp091-go"
)

func InitRunner(ctx context.Context, deps utils.EngineDeps) {

	/*
		tmp subdirectories are created in layers.
		The base ones (/tmp/runner/sockets /tmp/runner/testsets /tmp/runner/submissions ) are created during initiation (here)
		Slot specific sockets are created during warm container creation
		Submission specific subdirectories are created during orchestration (after container getting job)
	*/

	// base subdirectories ( /tmp/runner mounted as /workspace in all containers)
	dirs := []string{
		filepath.Join("/tmp", "runner", "sockets"),
		filepath.Join("/tmp", "runner", "testsets"),
		filepath.Join("/tmp", "runner", "submissions"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			log.Fatalf("failed to create base temp directories %s: %v", dir, err)
		}
	}

	// rmq consumer (to collect jobspecs from rmq)
	localqueue := make(chan amqp.Delivery)
	if err := deps.Rmq.Subscribe(ctx, localqueue, deps.JobQueue, "rmq-consoomer"); err != nil {
		log.Fatalf("Failed to initiate rmq consumer %v", err)
	}

	cCtx := namespaces.WithNamespace(ctx, deps.Namespace)

	// create warm continaers | conainerQueue is buffered with MAX_CONTAINER_CAP(15 for now) , if made unbuffered, runner will freeze shortly
	containerQueue := make(chan *internal.WarmContainer, 15)
	countContainer := 0
	var slotCounter atomic.Uint32 // Used atomic counter to prevent concurrency issues affecting slotID uniqueness

	// warm container producer loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				countContainer++
				log.Printf("Creating container %v", countContainer)

				// create the warm container
				slotID := slotCounter.Add(1)
				warmc, err := internal.CreateWarmContainer(cCtx, deps.Client, slotID)
				if err != nil {
					log.Printf("Warm container creation failure: %v", err)
					time.Sleep(1 * time.Second) // short cooldown
					continue
				}
				select {
				// send the warm container in containerQueue
				case containerQueue <- warmc:
				case <-ctx.Done():
					log.Printf("CRITICAL: Producer context canceled! Deleting container slot %d...", slotID)
					_ = warmc.Container.Delete(cCtx, containerd.WithSnapshotCleanup)
					return
				}

			}
		}
	}()

	// main orchestration
	var wg sync.WaitGroup
	for {

		select {
		case <-ctx.Done():
			log.Println("Shutting down runner engine")
			wg.Wait() // block until active OrchestrateSubm workers finish
			return
		case msg, ok := <-localqueue:
			if !ok {
				log.Println("Local queue channel closed, exiting")
				return
			}

			// worker go-routine
			wg.Add(1)
			go func(delivery amqp.Delivery) {
				defer wg.Done()

				var warmcontainer *internal.WarmContainer

				select {
				case warmcontainer = <-containerQueue:
					time.Sleep(50 * time.Millisecond) // small cooldown
				case <-time.After(3 * time.Second):

					// fallback on-demand container creation in case of queue overload
					slotID := slotCounter.Add(1)
					var err error
					warmcontainer, err = internal.CreateWarmContainer(cCtx, deps.Client, slotID)
					if err != nil {
						log.Printf("Failed to create contianer %d (on-demand): %v", slotID, err)
					}
				}

				jobspec, err := utils.ProcessJobSpec(ctx, msg, deps.SSEQueue)
				if err != nil {
					log.Printf("Jobspec parsing error: %v", err)
					_ = delivery.Nack(false, true)
				}

				contInfo, err := OrchestrateSubm(ctx, cCtx, warmcontainer, *deps.S3, jobspec, *deps.Rmq)
				if err != nil {
					log.Printf("Orchestrator error: %v", err)
				}
				_ = delivery.Ack(false) // send ACK to rmq
				fmt.Printf("Container info: %v", contInfo)

			}(msg)
		}
	}
}
