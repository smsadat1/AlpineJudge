package runner

import (
	"context"
	"local/runner/executor"
	"local/runner/scheduler"
	"log"
	"os"
	"sync/atomic"
	"time"

	containerd "github.com/containerd/containerd"
	namespaces "github.com/containerd/containerd/v2/pkg/namespaces"
	amqp "github.com/rabbitmq/amqp091-go"

	"shared"
	"utils"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Loading configurations...")
	if err := utils.LoadRunnerConfigs("/etc/alpinejudge/runner.yaml"); err != nil {
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

	sysMetrics := make(chan utils.SystemMetrics, 15)
	localQueue := make(chan amqp.Delivery, 100)
	containerQueue := scheduler.InitContainerQueue()

	log.Println("Initializing containerd client socket...")
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatalf("Failed to initiate containerd: %v", err)
	}
	defer client.Close()

	log.Println("Creating container namespace...")
	cCtx := namespaces.WithNamespace(ctx, "alpine_judge")

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

	var (
		currDecisions   utils.RADSDecision
		activeTasks     int32 // Atomic counter for active jobs running
		targetWarmCount int32 // Target warm containers to maintain
	)

	/*
		Warm Container Producer Loop
		Maintains dynamic pool size determined by RADScheduler decisions
	*/
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// If warm pool size is below target budget, create a pre-warmed container
				// TODO: add a channel length check or atomic count tracker for containerQueue
				// _ := atomic.LoadInt32(&targetWarmCount) (use this for that)

				warmContainer, err := executor.CreateWarmContainer(cCtx, client)
				if err != nil {
					log.Printf("Warm Pool Error: failed to create pre-warmed container: %v", err)
					time.Sleep(1 * time.Second)
					continue
				}

				// Push into dynamic container queue
				containerQueue.In <- warmContainer
			}
		}
	}()

	log.Println("Runner Daemon successfully initialized and monitoring...")

	// Main orchestration loop
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down Runner daemon loops gracefully...")
			return

		case sysm := <-sysMetrics:
			currentActive := int(atomic.LoadInt32(&activeTasks))
			currDecisions = scheduler.RADScheduler(sysm.AvailableMemoryMB, sysm.CPUCoreCount, currentActive) // assigned to outer variable

			// Update target warm pool size dynamically based on scheduler capacity
			atomic.StoreInt32(&targetWarmCount, int32(currDecisions.AvailableSlots))

			log.Printf("Resource Sync -> Slots Available: %d | Idle: %f | Running Tasks: %d\n",
				currDecisions.AvailableSlots, currDecisions.IdleSlots, currentActive)

		case msg, ok := <-localQueue:
			if !ok {
				log.Println("Critical: Local queue channel closed unexpectedly.")
				return
			}

			currActive := atomic.LoadInt32(&activeTasks)
			// Backpressue check
			if int(currActive) >= currDecisions.AvailableSlots || currDecisions.IdleSlots <= 0 {
				log.Printf("Backpressure Warning: Maximum scheduling slots reached (%d/%d). Re-queuing.",
					currActive, currDecisions.AvailableSlots)

				_ = msg.Nack(false, true)
				continue
			}

			// Reserve slot atomically
			atomic.AddInt32(&activeTasks, 1)

			go func(delivery amqp.Delivery) {
				// atomic decrement
				defer atomic.AddInt32(&activeTasks, -1)
				log.Printf("Allocating slot. Fetching pre-warmed container for message ID: %s\n", delivery.MessageId)

				var warmCont containerd.Container
				select {
				case warmCont = <-containerQueue.Out:
				case <-time.After(3 * time.Second):
					log.Println("Warm pool starved, falling back to on-demand creation...")
				}

				jobspec, err := utils.ProcessJobSpec(ctx, msg)
				if err != nil {
					log.Printf("Jobspec Parse Error: %v", err)
					_ = delivery.Nack(false, false)
					return
				}

				// Execute using warm container or fallback to on-demand
				contInfo, err := executor.OrchestrateSubm(ctx, warmCont, *s3m, jobspec, *rmqm, false)

				log.Printf("Container Info -> SubmissionID: %v | Status: %v", contInfo.SubmissionId, contInfo.Status)

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
