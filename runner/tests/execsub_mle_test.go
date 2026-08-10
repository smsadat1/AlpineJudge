package tests

import (
	"assert"
	"context"
	"encoding/json"
	"fmt"
	"local/runner/pkg"
	"testing"
	"time"
	"utils"

	"github.com/containerd/containerd/v2/pkg/namespaces"
	amqp "github.com/rabbitmq/amqp091-go"
)

func Test_ExecSubm_MLE(t *testing.T) {

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	ctx = namespaces.WithNamespace(ctx, "test-namespace")
	defer cancel()

	tf := pkg.NewRunnerTestFactory(t)
	tr := pkg.NewRunnerTestRepository(t)
	tr.TestAgentExecSpec.CompileArgs[5] = fmt.Sprintf("/workspace/submissions/%v/mlemaker.cpp", tr.TestJobSpec.SubmissionID)
	tr.CreateTempLocations(t)
	tr.CopyFiles(t, "../examples/mlemaker.cpp")

	tf.StartTestMinioS3(t, ctx)
	tf.StartTestRMQ(t, ctx)
	tf.GetWarmContainer(t, ctx)

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

				assert.String(t, "INFO", testEventStream.Type)
				assert.String(t, "Memory limit exceeded", testEventStream.Status)
			}
		}
	}()

	// channel to safely receive the return value of ExecSubm
	type result struct {
		info utils.ContainerInfo
	}
	execChan := make(chan result, 1)

	go func() {
		info := tf.WarmContainer.ExecSubm(ctx, tr.TestExecRules, tr.TestJobSpec, *tf.Rmqm, *tf.S3m)
		execChan <- result{info: info}
	}()

	// send execspec
	if err := json.NewEncoder(tf.WarmContainer.Conn).Encode(&tr.TestAgentExecSpec); err != nil {
		t.Errorf("Failed sending execspec JSON: %v", err)
	}

	// wait for ExecSubm() to complete
	var contInfo utils.ContainerInfo
	select {
	case result := <-execChan:
		contInfo = result.info
	case <-ctx.Done():
		t.Error("Timed out waiting for execution submission")
	}

	<-eventsDone

	assert.String(t, tr.TestJobSpec.SubmissionID, contInfo.SubmissionId)
	assert.String(t, tr.TestJobSpec.Language, contInfo.Language)

	t.Logf(`Container lifecycle data
			Elapsed time: %vms | Status: %v | StatusInfo: %v
			Stderr: %v
			Stdout: %v`,
		contInfo.Interval, contInfo.Status, contInfo.StatusInfo, contInfo.ContainerStderr, contInfo.ContainerStdout)
}
