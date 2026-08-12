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

	"github.com/containerd/containerd/v2/pkg/namespaces"
	amqp "github.com/rabbitmq/amqp091-go"
)

func Test_Orchestrator(t *testing.T) {

	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	cCtx := namespaces.WithNamespace(ctx, "test-namespace")
	defer cancel()

	tr := pkg.NewRunnerTestRepository(t)
	// tf := pkg.NewRunnerTestFactory(t)
	// tf.StartTestMinioS3(t, ctx)
	// tf.StartTestRMQ(t, ctx)
	warmCont := SharedTF.GetWarmContainer(t, cCtx)

	// upload artifacts to S3 first ==============
	srcFileData, err := os.Open("../examples/main.cpp")
	if err != nil {
		t.Fatalf("Failed to get source submission file: %v", err)
	}
	if err := SharedTF.S3m.UploadFileToS3(ctx, tr.TestJobSpec.SrcCodeS3Key, srcFileData); err != nil {
		t.Fatalf("Failed to upload source file: %v", err)
	}
	if err := SharedTF.S3m.UploadDirToS3(ctx, tr.TestJobSpec.TestsetS3Key, "../examples/ts001"); err != nil {
		t.Fatalf("Failed to upload testsests: %v", err)
	}

	// get events from RMQ
	interceptorQueue := make(chan amqp.Delivery, 100)
	if err := SharedTF.Rmqm.Subscribe(ctx, interceptorQueue, tr.TestJobSpec.SSEQueue, "test-consoomer"); err != nil {
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
	execChan := make(chan result, 1)

	go func() {
		info, err := pkg.OrchestrateSubm(ctx, cCtx, warmCont, *SharedTF.S3m, tr.TestJobSpec, *SharedTF.Rmqm)
		execChan <- result{info: info, err: err}
	}()

	// wait for OrchestrateSubm() to complete
	var contInfo utils.ContainerInfo
	var orchError string
	select {
	case result := <-execChan:
		contInfo = result.info
		orchError = fmt.Sprintf("%v", result.err)
	case <-ctx.Done():
		t.Error("Timed out waiting for execution submission")
	}

	<-eventsDone

	t.Logf(`Container lifecycle data
			SubmissionID: %v
			Language: %v
			Version: %v
			Elapsed time: %vms
			Status: %v
			StatusInfo: %v
			Stderr: %v
			Stdout: %v
			Orhcestration error: %v`,
		contInfo.SubmissionId, contInfo.Language, contInfo.Version,
		contInfo.Interval, contInfo.Status, contInfo.StatusInfo, contInfo.ContainerStderr, contInfo.ContainerStdout,
		orchError)

}
