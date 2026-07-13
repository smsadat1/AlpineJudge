package dispatcher

import (
	"context"
	"log"
	"os"
	"shared"

	"github.com/joho/godotenv"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := godotenv.Load("/etc/alpinejudge/dispatcher/dispatcher.env"); err != nil {
		log.Fatal("Failed to load dispatcher env vars\n")
	}

	log.Println("Loading dispatcher configuration")
	if err := LoadConfigs("/etc/alpinejudge/dispatcher/config.yaml"); err != nil {
		log.Fatalf("%v", err)
	}

	// initiate s3
	bucket := os.Getenv("MINIO_S3_BUCKET")
	region := os.Getenv("S3_REGION_NAME")
	accessKey := os.Getenv("S3_USERNAME_DEV")
	secretKey := os.Getenv("S3_PASSWORD_DEV")
	s3Endpoint := os.Getenv("S3_ENDPOINT_DEV")

	// s3Mgr, err := shared.InitS3Manager(ctx, bucket, region, accessKey, secretKey, s3Endpoint)
	_, err := shared.InitS3Manager(ctx, bucket, region, accessKey, secretKey, s3Endpoint)
	if err != nil {
		log.Fatalf("Failed to spin up S3: %v", err)
	}

	// start rmq channel
	amqpURL := os.Getenv("RABBITMQ_URL_DEV")
	if amqpURL == "" {
		log.Fatal("RMQ url not found in environment!\n")
	}

	rmqMgr, err := shared.NewRMQManager(ctx, amqpURL)
	if err != nil {
		log.Fatalf("Failed to spin up RabbitMQ: %v", err)
	}
	defer rmqMgr.Close()

	// start http server
	// InitHttpServer(ctx, s3Mgr, rmqMgr)
}
