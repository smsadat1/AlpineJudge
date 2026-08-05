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
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		t.Fatalf("Failed connecting containerd client: %v", err)
	}

	tr := pkg.NewRunnerTestRepository(t)
	tf := pkg.NewRunnerTestFactory(t)
	tf.StartTestMinioS3(t, ctx)
	tf.StartTestRMQ(t, ctx)

	// upload artifacts to S3 first ==============
	srcFileData, err := os.Open("../examples/main.cpp")
	if err != nil {
		t.Fatalf("Failed to get source submission file: %v", err)
	}
	if err := tf.S3m.UploadFileToS3(ctx, tr.TestJobSpec.SrcCodeS3Key, srcFileData); err != nil {
		t.Fatalf("Failed to upload source file: %v", err)
	}
	if err := tf.S3m.UploadDirToS3(ctx, tr.TestJobSpec.TestsetS3Key, "../examples/ts001"); err != nil {
		t.Fatalf("Failed to upload testsests: %v", err)
	}

	// get events from RMQ
	interceptorQueue := make(chan amqp.Delivery, 100)
	if err := tf.Rmqm.Subscribe(ctx, interceptorQueue, tr.TestJobSpec.SSEQueue, "test-consoomer"); err != nil {
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
				assert.String(t, "INFO", testEventStream.Type)
				assert.String(t, fmt.Sprintf("Running test %v", counter), testEventStream.Status)
			}
		}
	}()

	// channel to safely receive the return value of ExecSubm
	type result struct {
		info utils.ContainerInfo
		err  error
	}
	execChan := make(chan error, 1)

	// send jobspec payload over rmq
	jobPayload, err := json.Marshal(tr.TestJobSpec)
	if err != nil {
		t.Fatalf("Failed marshaling jobspec payload: %v", err)
	}
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        jobPayload,
	}
	if err := tf.Rmqm.Publish(ctx, "later", msg); err != nil {
		t.Fatalf("Failed sending jobspec payload over rmq: %v", err)
	}

	deps := utils.EngineDeps{
		Client:    client,
		Rmq:       tf.Rmqm,
		S3:        tf.S3m,
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

	go func() {
		pkg.InitRunner(ctx, deps)
		execChan <- err
	}()

	<-eventsDone
}
