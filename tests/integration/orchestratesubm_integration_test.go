package integration

import (
	"assert"
	"context"
	"encoding/json"
	"fmt"
	"local/runner/executor"
	"local/testrunner/factory"
	"local/testrunner/repository"
	"os"
	"testing"
	"time"
	"utils"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	amqp "github.com/rabbitmq/amqp091-go"
)

func Test_OrchestrateSubm_integration_test(t *testing.T) {

	ctx, cancel := context.WithTimeout(
		namespaces.WithNamespace(context.Background(), "test-namespace"),
		45*time.Second,
	)
	defer cancel()

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		t.Skipf("Skipping integration test: local containerd socket not available: %v", err)
	}
	defer client.Close()

	tf := factory.NewTestFactory(t)
	tf.StartTestMinioS3(t, ctx)
	tf.StartTestRMQ(t, ctx)

	tr := repository.NewTestRepository(t)

	// load configs
	if err := utils.LoadRunnerConfigs("../artifacts/runner.config.example.yaml"); err != nil {
		t.Fatalf("Failed to load configs: %v", err)
	}

	// upload in s3 first for test
	srcFileData, _ := os.OpenFile("../artifacts/main.cpp", os.O_RDONLY, os.FileMode(os.O_APPEND))
	if err := tf.S3m.UploadFileToS3(ctx, tr.TestJobSpec.SrcCodeS3Key, srcFileData); err != nil {
		t.Fatalf("Failed uploading file to S3: %v", err)
	}
	if err := tf.S3m.UploadDirToS3(ctx, tr.TestJobSpec.TestsetS3Key, "../artifacts/ts001"); err != nil {
		t.Fatalf("Failed uploading directory to S3: %v", err)
	}

	// get events from RMQ
	interceptorQueue := make(chan amqp.Delivery, 100)
	collectMesg := make(chan string, 100)
	if err := tf.Rmqm.Subscribe(ctx, interceptorQueue, tr.TestJobSpec.EventQueue, "consoomer"); err != nil {
		t.Fatalf("Failed to start rmq consumer: %v", err)
	}

	// Goroutine reading events cleanly with context cancellation
	go func() {
		counter := 0
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-interceptorQueue:
				if !ok {
					return
				}
				_ = delivery.Ack(false)

				var testEventStream utils.Event
				json.Unmarshal(delivery.Body, &testEventStream)

				counter++
				assert.String(t, testEventStream.Type, "INFO")
				assert.String(t, testEventStream.Status, fmt.Sprintf("Running test %v", counter))

				select {
				case collectMesg <- string(delivery.Body):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	result, err := executor.OrchestrateSubm(ctx, client, *tf.S3m, tr.TestJobSpec, *tf.Rmqm, false)
	if err != nil {
		t.Errorf("Failed to get test result from orchestrator: %v", err)
	}

	select {
	case msg, ok := <-collectMesg:
		if !ok {
			t.Errorf("Failed to capture live status event: %s", msg)
		}
	case <-ctx.Done():
		// t.Fatal("Timed out waiting for live status message on RMQ")
	}

	t.Logf(`Container lifecycle data
			SubmissionID: %v
			Language: %v
			Version: %v
			Elapsed time: %vms
			Status: %v
			StatusInfo: %v
			Stderr: %v
			Stdout: %v`,
		result.SubmissionId, result.Language, result.Version, result.Interval,
		result.Status, result.StatusInfo, result.ContainerStderr, result.ContainerStdout)
}
