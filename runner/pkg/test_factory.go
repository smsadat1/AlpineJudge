package pkg

import (
	"context"
	"fmt"
	"local/runner/internal"
	"log"
	"os"
	"shared"
	"testing"

	containerd "github.com/containerd/containerd"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

type TestFactory struct {
	s3URL          string
	s3UserName     string
	s3Password     string
	S3bucket       string
	s3Region       string
	minioContainer *minio.MinioContainer
	S3m            *shared.S3Manager

	rmqURL       string
	RmqQueueName string
	rmqContainer *rabbitmq.RabbitMQContainer
	Rmqm         *shared.RMQManager

	Image         string
	WarmContainer *internal.WarmContainer
}

func NewRunnerTestFactory() *TestFactory {

	// t.Helper()

	s3Bucket := os.Getenv("TEST_S3_BUCKET_NAME")
	s3Region := os.Getenv("TEST_S3_REGION_NAME")
	s3UserName := os.Getenv("TEST_S3_USERNAME")
	s3Password := os.Getenv("TEST_S3_PASSWORD")
	s3URL := os.Getenv("TEST_S3_URL")
	rmqURL := os.Getenv("TEST_RABBITMQ_URL")
	rmqq := os.Getenv("RABBITMQ_QUEUE_NAME")

	return &TestFactory{
		s3URL:        s3URL,
		S3bucket:     s3Bucket,
		s3UserName:   s3UserName,
		s3Password:   s3Password,
		s3Region:     s3Region,
		rmqURL:       rmqURL,
		RmqQueueName: rmqq,
		Image:        "ghcr.io/smsadat1/alpinejudge/gcc:test",
	}
}

func (tf *TestFactory) StartTestRMQ(ctx context.Context) {

	// t.Helper()

	rmqContainer, err := rabbitmq.Run(
		ctx,
		"rabbitmq:3.12.11-management-alpine",
		rabbitmq.WithAdminUsername("guest"),
		rabbitmq.WithAdminPassword("guest"),
		testcontainers.WithExposedPorts("5672"),
	)
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	amqpURL, err := rmqContainer.AmqpURL(ctx)
	if err != nil {
		log.Fatalf("failed to get amqp url: %v", err)
	}

	tf.rmqContainer = rmqContainer
	tf.rmqURL = amqpURL
	tf.Rmqm, err = shared.NewRMQManager(ctx, tf.rmqURL)

	if err != nil {
		log.Fatalf("failed to setup rabbitmq manager: %v", err)
	}
}

func (tf *TestFactory) StartTestMinioS3(ctx context.Context) {

	// t.Helper()
	// Command to configure alias, create bucket, and set region using internal 'mc' tool
	setupCmd := testcontainers.NewRawCommand([]string{
		"sh", "-c",
		fmt.Sprintf(
			"mc alias set myminio http://localhost:9000 %s %s && mc mb --region=%s myminio/%s",
			tf.s3UserName, tf.s3Password, tf.s3Region, tf.S3bucket,
		),
	})

	minioContainer, err := minio.Run(
		ctx,
		"minio/minio:RELEASE.2024-01-16T16-07-38Z",
		minio.WithPassword(tf.s3Password),
		minio.WithUsername(tf.s3UserName),
		testcontainers.WithEnv(map[string]string{
			"MINIO_REGION_NAME": tf.s3Region, // set default S3 region
		}),
		testcontainers.WithAfterReadyCommand(setupCmd),
	)

	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	// Save host & port endpoint for your MinIO Go client
	endpoint, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get minio connection string: %v", err)
	}

	tf.minioContainer = minioContainer
	tf.s3URL = fmt.Sprintf("http://%s", endpoint)
	tf.S3m, err = shared.InitS3Manager(ctx, tf.S3bucket, tf.s3Region, tf.s3UserName, tf.s3Password, tf.s3URL)

	if err != nil {
		log.Fatalf("failted to setup S3 manager: %v", err)
	}
}

func (tf *TestFactory) GetWarmContainer(t *testing.T, ctx context.Context) *internal.WarmContainer {

	t.Helper()

	var testSlotID uint32
	testSlotID = 777

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		t.Skipf("Skipping integration test: local containerd socket not available: %v", err)
	}
	t.Cleanup(func() {
		client.Close()
	})

	wc, err := internal.CreateWarmContainer(ctx, client, testSlotID)
	if err != nil {
		t.Fatalf("Failed to get warm container: %v", err)
	}

	return wc
}

func (tf *TestFactory) CleanupGlobal(ctx context.Context) {

	if err := testcontainers.TerminateContainer(tf.rmqContainer); err != nil {
		log.Printf("failed to terminate rabbitmq container: %s", err)
	}

	if err := testcontainers.TerminateContainer(tf.minioContainer); err != nil {
		log.Printf("failed to terminate minio container: %s", err)
	}
}
