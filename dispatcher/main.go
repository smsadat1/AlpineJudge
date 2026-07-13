package dispatcher

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"shared"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	/*
		Signal aware context
		Auto cancels when the OS sends termination commands like Ctrl+C (SIGINT) or systemd stop (SIGTERM).
	*/
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. envs & cofigs ingestion
	if err := godotenv.Load("dispatcher.example.env"); err != nil {
		log.Fatal("Failed to load dispatcher env vars\n")
	}

	log.Println("Loading dispatcher configurations...")
	if err := LoadConfigs("config.example.yaml"); err != nil {
		log.Fatalf("%v", err)
	}

	// 2. initialize infrastructures
	log.Println("Initiating S3 storage...")
	bucket := os.Getenv("MINIO_S3_BUCKET")
	region := os.Getenv("S3_REGION_NAME")
	accessKey := os.Getenv("S3_USERNAME_DEV")
	secretKey := os.Getenv("S3_PASSWORD_DEV")
	s3Endpoint := os.Getenv("S3_ENDPOINT_DEV")

	s3m, err := shared.InitS3Manager(ctx, bucket, region, accessKey, secretKey, s3Endpoint)
	if err != nil {
		log.Fatalf("Failed to spin up S3: %v", err)
	}

	log.Println("Initiating RMQ connection...s")
	amqpURL := os.Getenv("RABBITMQ_URL_DEV")
	if amqpURL == "" {
		log.Fatal("RMQ url not found in environment!\n")
	}

	rmqMgr, err := shared.NewRMQManager(ctx, amqpURL)
	if err != nil {
		log.Fatalf("Failed to spin up RabbitMQ: %v", err)
	}
	defer func() {
		log.Println("Closing RabbitMQ sockets...")
		rmqMgr.Close()
	}()

	log.Println("Starting Dispatcher HTTP server...")
	server := InitHTTPServer(ctx, s3m, rmqMgr)

	// 3. background HTTP server listener to make it non-blocking
	go func() {
		log.Printf("Dispatcher listening securely on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Critical HTTP server crash: %v", err)
		}
	}()

	// wait for OS signal to stop
	<-ctx.Done()
	log.Println("Termination signal caught! Initiating graceful teardown protocol...")

	// 4. raceful shutdown Phase
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
