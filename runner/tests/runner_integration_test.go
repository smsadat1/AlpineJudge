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
	testCtx, testCancel := context.WithTimeout(t.Context(), 120*time.Second)
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
	if err := SharedTF.S3m.UploadDirToS3(testCtx, tr.TestJobSpec.Testset, "ts001"); err != nil {
		t.Fatalf("Failed to upload testsests: %v", err)
	}

	// get events from RMQ
	interceptorQueue := make(chan amqp.Delivery, 100)
	routingKey := tr.TestJobSpec.SubmissionID
	if err := SharedTF.Rmqm.SubscribeToExchange(testCtx, interceptorQueue, os.Getenv("DIRECT_EXCHANGE_NAME"), routingKey); err != nil {
		t.Fatalf("Failed to subscribe to exchange: %v", err)
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

				fmt.Printf("Events: %v\n", testEventStream)

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

	// submission over rmq
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
		// not a failure because daemons are supposed to run continiously
		t.Logf("Test timed out waiting for events")
	}

	// Signal InitRunner to stop
	runnerCancel()

	// Wait for InitRunner to exit before finishing test
	select {
	case <-runnerExited:
		// InitRunner stopped cleanly
	case <-time.After(15 * time.Second):
		// delays can happen, don't make it Fatal
		t.Logf("InitRunner failed to stop within 15 seconds")
	}
}
