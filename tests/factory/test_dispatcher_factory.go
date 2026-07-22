package factory

import (
	"context"
	"fmt"
	"log"
	"os"
	"shared"
	"testing"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/platforms"
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

	Image string
}

func NewTestFactory(t *testing.T) *TestFactory {

	t.Helper()

	t.Setenv("TEST_S3_URL", "http://localhost:9000")
	t.Setenv("TEST_S3_USERNAME", "minioadmin")
	t.Setenv("TEST_S3_PASSWORD", "minioadminpassword")
	t.Setenv("TEST_S3_BUCKET_NAME", "ajbucket-test-e2e-d")
	t.Setenv("TEST_S3_REGION_NAME", "us-east-1")
	t.Setenv("TEST_RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	t.Setenv("RABBITMQ_QUEUE_NAME", "job-queue-runner1")

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

func (tf *TestFactory) StartTestRMQ(t *testing.T, ctx context.Context) {

	t.Helper()

	declareQueueCmd := testcontainers.NewRawCommand([]string{
		"rabbitmqadmin",
		"declare",
		"queue",
		fmt.Sprintf("name=%s", tf.RmqQueueName),
		"durable=true",
	})

	rmqContainer, err := rabbitmq.Run(
		ctx,
		"rabbitmq:3.12.11-management-alpine",
		rabbitmq.WithAdminUsername("guest"),
		rabbitmq.WithAdminPassword("guest"),
		testcontainers.WithExposedPorts("5672"),
		testcontainers.WithAfterReadyCommand(declareQueueCmd),
	)

	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	// Register teardown with Go test framework
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(rmqContainer); err != nil {
			log.Printf("failed to terminate rabbitmq container: %s", err)
		}
	})

	amqpURL, err := rmqContainer.AmqpURL(ctx)
	if err != nil {
		t.Fatalf("failed to get amqp url: %v", err)
	}

	tf.rmqContainer = rmqContainer
	tf.rmqURL = amqpURL
	tf.Rmqm, err = shared.NewRMQManager(ctx, tf.rmqURL)

	if err != nil {
		t.Fatalf("failed to setup rabbitmq manager: %v", err)
	}
}

func (tf *TestFactory) StartTestMinioS3(t *testing.T, ctx context.Context) {

	t.Helper()
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
		t.Fatalf("failed to start container: %s", err)
	}

	// Clean up container automatically when the test finishes
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(minioContainer); err != nil {
			log.Printf("failed to terminate minio container: %s", err)
		}
	})

	// Save host & port endpoint for your MinIO Go client
	endpoint, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get minio connection string: %v", err)
	}

	tf.minioContainer = minioContainer
	tf.s3URL = fmt.Sprintf("http://%s", endpoint)
	tf.S3m, err = shared.InitS3Manager(ctx, tf.S3bucket, tf.s3Region, tf.s3UserName, tf.s3Password, tf.s3URL)

	if err != nil {
		t.Fatalf("failted to setup S3 manager: %v", err)
	}
}

func (tf *TestFactory) StartRawContainer(t *testing.T, ctx context.Context) containerd.Container {
	ctx = namespaces.WithNamespace(ctx, "test")

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		t.Fatalf("failed to connect to containerd: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	targetPlatform := platforms.DefaultString()
	image, err := client.Pull(
		ctx,
		tf.Image,
		containerd.WithPullUnpack,
		containerd.WithPullSnapshotter("native"),
		containerd.WithPlatform(targetPlatform),
	)
	if err != nil {
		t.Fatalf("failed to pull image %s: %v", tf.Image, err)
	}

	// Dynamic unique IDs to prevent state collision on disk
	containerID := fmt.Sprintf("test-runner-%d", time.Now().UnixNano())
	snapshotID := containerID + "-snapshot"

	container, err := client.NewContainer(
		ctx,
		containerID,
		// ORDER MATTERS: Set native snapshotter BEFORE telling it to build the snapshot!
		containerd.WithSnapshotter("native"),
		containerd.WithNewSnapshot(snapshotID, image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	// Cleanup container and its snapshot after test completes
	t.Cleanup(func() {
		_ = container.Delete(ctx, containerd.WithSnapshotCleanup)
	})

	return container
}
