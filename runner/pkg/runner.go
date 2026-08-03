package pkg

import (
	"context"
	"log"
	"os"
	"shared"
	"utils"

	"github.com/containerd/containerd"
	"github.com/joho/godotenv"
)

/*
The top-most level of the Runner subsystem. Gets called by main()
It runs as daemon process so it's never intended to exit unless crashes internally or Linux kernel intervense
*/
func Runner() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	godotenv.Load(".env")

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

	edps := utils.EngineDeps{
		Client:    client,
		S3:        s3m,
		Rmq:       rmqm,
		JobQueue:  os.Getenv("RABBITMQ_JOB_QUEUE_NAME"),
		SSEQueue:  os.Getenv("RABBITMQ_SSE_QUEUE_NAME"),
		Namespace: os.Getenv("CONTAINER_NAMESPACE"),
	}

	InitRunner(ctx, edps)
}
