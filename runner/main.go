/*
*  Check the system
*  Connect to RMQ -> initiate RMQ consumer
*  Pull images & cache
*  Start pulling job specs from RMQ global queue
 */

package runner

import (
	"context"
	"local/runner/executor"
	"local/runner/images"
	"local/runner/scheduler"
	"local/runner/utils"
	"log"
	"os"
	"shared"
	"time"

	containerd "github.com/containerd/containerd"
	namespaces "github.com/containerd/containerd/v2/pkg/namespaces"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Loading configurations...")
	if err := utils.LoadRunnerConfigs("config.example.yaml"); err != nil {
		log.Fatalf("Fatal: Configuration failed to load: %v", err)
	}

	rmqm, err := shared.NewRMQManager(ctx, os.Getenv("RABBITMQ_URL_DEV"))
	if err != nil {
		log.Fatalf("Fatal: RabbitMQ connection broken: %v", err)
	}
	defer rmqm.Close()

	log.Println("Initializing S3...")
	s3m, err := shared.InitS3Manager(
		ctx,
		os.Getenv("S3_BUCKET_NAME"),
		os.Getenv("S3_REGION_NAME"),
		os.Getenv("S3_USERNAME_DEV"),
		os.Getenv("S3_PASSWORD_DEV"),
		os.Getenv("S3_URL_DEV"),
	)
	if err != nil {
		log.Fatalf("Fatal: S3 Storage initialization aborted: %v", err)
	}

	log.Println("Initializing containerd client socket...")
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatalf("Failed to initiate containerd: %v", err)
	}
	defer client.Close()

	log.Println("Creating container namespace...")
	cCtx := namespaces.WithNamespace(ctx, "alpine_judge")

	images.EnsureContainerImages()

	sysMetrics := make(chan utils.SystemMetrics, 15)
	localQueue := make(chan amqp.Delivery, 100)

	log.Println("Firing background workers (SystemMonitor & RabbitMQ consumer)...")
	go func() {
		err = scheduler.SystemMonitor(ctx, time.Duration(15)*time.Second, sysMetrics)
		if err != nil {
			log.Printf("Telemetry Alert: SystemMonitor routine collapsed: %v\n", err)
		}
	}()
	go func() {
		if err := rmqm.Subscribe(ctx, localQueue, "job-queue-consumer", "runner001-consumer"); err != nil {
			log.Printf("Broker Alert: Consumer subscription severed: %v\n", err)
		}
	}()

	log.Println("Runner Daemon successfully initialized and monitoring...")

	// main orchestration Loop
	var currentDecisions utils.RADSDecision
	runningContainers := 0

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down Runner daemon loops gracefully...")
			return

		case sysm := <-sysMetrics:
			currentDecisions := scheduler.RADScheduler(sysm.AvailableMemoryMB, sysm.CPUCoreCount, runningContainers)
			log.Printf("Resource Sync -> Slots Available: %d | Idle: %f | Running Tasks: %d\n",
				currentDecisions.AvailableSlots, currentDecisions.IdleSlots, runningContainers)

		case msg, ok := <-localQueue:
			if !ok {
				log.Println("Critical: Local queue channel closed unexpectedly.")
				return
			}

			// decide if physical runtime slots left based on the last telemetry check
			if runningContainers >= currentDecisions.AvailableSlots || currentDecisions.IdleSlots <= 0 {
				log.Printf("Backpressure Warning: Maximum scheduling slots reached (%d/%d). Rejecting/Re-queuing event.",
					runningContainers, currentDecisions.AvailableSlots)

				// Nack the message and throw it back onto RabbitMQ so another worker can take it
				_ = msg.Nack(false, true)
				continue
			}

			// while there's slot budget
			runningContainers++
			currentDecisions.IdleSlots--

			go func(delivery amqp.Delivery) {
				// safely decrement running tracker when container terminates execution
				defer func() {
					runningContainers--
				}()

				log.Printf("Allocating slot. Launching isolation runtime for message ID: %s\n", delivery.MessageId)

				jobspec, err := utils.ProcessJobSpec(ctx, msg)
				result, err := executor.ExecSubmission(cCtx, client, *s3m, jobspec, *rmqm)
				_ = result
				if err != nil {
					log.Printf("Execution Failure: Container run errored out: %v", err)
					_ = delivery.Nack(false, false) // drop bad tasks (add DLQ here)
					return
				}

				_ = delivery.Ack(false) // notify RMQ that task is cleared
				log.Printf("Task successfully executed. Slot released.")
			}(msg)
		}
	}
}
