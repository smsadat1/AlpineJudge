package tests

import (
	"context"
	"local/runner/pkg"
	"os"
	"testing"
)

// global shared test objects
var SharedTF *pkg.TestFactory

/*
Yes, many tests in this group can be paralleled but it gives little ROI
Tests of these group are all about whether a singla pass in the pipeline works as is
Where parallelism reduces chances to determine that introduces undesired race conditions to test itsel
Requiring extra effort to handle race conditions that aren't even part of the engine itself
*/
func TestMain(m *testing.M) {
	ctx := context.Background()

	os.Setenv("TEST_S3_URL", "http://localhost:9000")
	os.Setenv("TEST_S3_USERNAME", "minioadmin")
	os.Setenv("TEST_S3_PASSWORD", "minioadminpassword")
	os.Setenv("TEST_S3_BUCKET_NAME", "testbucket")
	os.Setenv("TEST_S3_REGION_NAME", "us-east-1")
	os.Setenv("TEST_RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	os.Setenv("RABBITMQ_QUEUE_NAME", "queue-001")

	os.Setenv("CONTAINER_QUEUECAP", "10")
	os.Setenv("TIMEOUT_SEC", "300")
	os.Setenv("MEMORY_LIMIT_MB", "4096")
	os.Setenv("PID_LIMIT", "128")
	os.Setenv("MAX_LOG_CAP_KB", "460800")
	os.Setenv("CPU_QUOTA", "2.0")
	os.Setenv("NO_NEW_PRIVILEGES", "true")
	os.Setenv("READONLY_ROOTFS", "true")

	os.Setenv("DIRECT_EXCHANGE_NAME", "ajxchnage")

	tf := pkg.NewRunnerTestFactory()
	tf.StartTestMinioS3(ctx)
	tf.StartTestRMQ(ctx)

	SharedTF = tf

	code := m.Run()
	tf.CleanupGlobal(ctx)
	os.Exit(code)
}
