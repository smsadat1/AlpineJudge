package pkg

import (
	"context"
	"dispatcher/internal"

	// "dispatcher/internal"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"shared"
	"syscall"
	"time"
)

func Dispatcher() {
	/*
		Signal aware context
		Auto cancels when the OS sends termination commands like Ctrl+C (SIGINT) or systemd stop (SIGTERM).
	*/
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. initialize infrastructures
	log.Println("Initiating S3 storage...")
	bucket := os.Getenv("MINIO_S3_BUCKET")
	region := os.Getenv("MINIO_S3_REGION_NAME")
	accessKey := os.Getenv("MINIO_S3_USERNAME")
	secretKey := os.Getenv("MINIO_S3_PASSWORD")
	s3Endpoint := os.Getenv("MINIO_S3_API")

	s3m, err := shared.InitS3Manager(ctx, bucket, region, accessKey, secretKey, s3Endpoint)
	if err != nil {
		log.Fatalf("Failed to spin up S3: %v", err)
	}

	log.Println("Initiating RMQ connection...s")
	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		log.Fatal("RMQ url not found in environment!\n")
	}

	log.Printf("S3 bucket: %v | region: %v | accesskey: %v | secretkey: %v | S3 endpoint: %v | RMQ: %v\n",
		bucket, region, accessKey, secretKey, s3Endpoint, amqpURL)

	rmqMgr, err := shared.NewRMQManager(ctx, amqpURL)
	if err != nil {
		log.Fatalf("Failed to spin up RabbitMQ: %v", err)
	}
	defer func() {
		log.Println("Closing RabbitMQ sockets...")
		rmqMgr.Close()
	}()

	log.Println("Starting Dispatcher HTTP server...")
	server := internal.InitHTTPServer(ctx, s3m, rmqMgr)

	// 2. background HTTP server listener to make it non-blocking
	go func() {
		log.Printf("Dispatcher listening securely on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Critical HTTP server crash: %v", err)
		}
	}()

	// wait for OS signal to stop
	<-ctx.Done()
	log.Println("Termination signal caught! Initiating graceful teardown protocol...")

	// 3. raceful shutdown Phase
	// Force-kill the HTTP engine if it takes longer than 5 seconds to clear out pending traffic
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown warning: forced termination executed: %v", err)
	} else {
		log.Println("HTTP server closed cleanly.")
	}

	log.Println("Dispatcher daemon terminated cleanly")
}
