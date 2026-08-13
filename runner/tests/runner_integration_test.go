package tests

import (
	"assert"
	"context"
	"encoding/json"
	"fmt"
	"local/runner/pkg"
	"os"
	"testing"
	"time"
	"utils"

	"github.com/containerd/containerd"
	amqp "github.com/rabbitmq/amqp091-go"
)

func Test_InitRunner(t *testing.T) {
	testCtx, testCancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer testCancel()

	runnerctx, runnerCancel := context.WithCancel(testCtx)
	t.Cleanup(func() {
		defer runnerCancel() // to guarantee InitRunner shuts down when test finishes and prevent race conditions
	})

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		t.Fatalf("Failed connecting containerd client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	tr := pkg.NewRunnerTestRepository(t)

	// upload artifacts to S3 first ==============
	srcFileData, err := os.Open("../examples/main.cpp")
	if err != nil {
		t.Fatalf("Failed to get source submission file: %v", err)
	}
	if err := SharedTF.S3m.UploadFileToS3(testCtx, tr.TestJobSpec.SrcCodeS3Key, srcFileData); err != nil {
		t.Fatalf("Failed to upload source file: %v", err)
	}
	if err := SharedTF.S3m.UploadDirToS3(testCtx, tr.TestJobSpec.TestsetS3Key, "../examples/ts001"); err != nil {
		t.Fatalf("Failed to upload testsests: %v", err)
	}

	// get events from RMQ
	interceptorQueue := make(chan amqp.Delivery, 100)
	if err := SharedTF.Rmqm.Subscribe(testCtx, interceptorQueue, tr.TestJobSpec.SSEQueue, "test-consoomer"); err != nil {
		t.Fatalf("Failed to subscribe to queue: %v", err)
	}

	// channel to signal event end
	eventsDone := make(chan struct{})

	// Goroutine reading events cleanly with context cancellation
	go func() {
		defer close(eventsDone)
		counter := 0
		for {
			select {
			case <-testCtx.Done():
				return
			case delivery, ok := <-interceptorQueue:
				if !ok {
					return
				}
				_ = delivery.Ack(false)

				var testEventStream utils.Event
				json.Unmarshal(delivery.Body, &testEventStream)

				if testEventStream.Type == "RESULT" {
					assert.String(t, "Accepted", testEventStream.Status)
					break // Type RESULT means stream has ended
				}

				counter++
				assert.String(t, "INFO", testEventStream.Type)
				assert.String(t, fmt.Sprintf("Running test %v", counter), testEventStream.Status)

				if counter == 1 {
					/*
						InitRunner starts runner daemon service
						Daemon aren't intended to exit like other programs rather stays online till Kernel intervention or admin actions
						So just return when the desired assertion passed
					*/
					return
				}
			}
		}
	}()

	// send jobspec payload over rmq
	jobPayload, err := json.Marshal(tr.TestJobSpec)
	if err != nil {
		t.Fatalf("Failed marshaling jobspec payload: %v", err)
	}
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        jobPayload,
	}
	if err := SharedTF.Rmqm.Publish(testCtx, "later", msg); err != nil {
		t.Fatalf("Failed sending jobspec payload over rmq: %v", err)
	}

	deps := utils.EngineDeps{
		Client:    client,
		Rmq:       SharedTF.Rmqm,
		S3:        SharedTF.S3m,
		JobQueue:  "later",
		SSEQueue:  tr.TestJobSpec.SSEQueue,
		Namespace: "test-runner",
	}

	t.Setenv("READONLY_ROOTFS", "true")
	t.Setenv("NO_NEW_PRIVILEGES", "true")
	t.Setenv("CPU_QUOTA", "2.0")
	t.Setenv("MEMORY_LIMIT_MB", "1024")
	t.Setenv("PID_LIMIT", "128")
	t.Setenv("TIMEOUT_SEC", "300")

	runnerExited := make(chan struct{})
	go func() {
		defer close(runnerExited)
		pkg.InitRunner(runnerctx, deps)
	}()

	select {
	case <-eventsDone:
		// Events received successfully
	case <-testCtx.Done():
		t.Fatal("Test timed out waiting for events")
	}

	// Signal InitRunner to stop
	runnerCancel()

	// Wait for InitRunner to exit before finishing test
	select {
	case <-runnerExited:
		// InitRunner stopped cleanly
	case <-time.After(5 * time.Second):
		t.Fatal("InitRunner failed to stop within 5 seconds")
	}
}
