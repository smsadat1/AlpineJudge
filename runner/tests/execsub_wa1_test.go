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

// test continues even when WA happends
func Test_ExecSubm_WA(t *testing.T) {

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	ctx = namespaces.WithNamespace(ctx, "test-namespace")
	defer cancel()

	tr := pkg.NewRunnerTestRepository(t)
	tr.TestAgentExecSpec.CompileArgs[5] = fmt.Sprintf("/workspace/submissions/%v/wrongans.cpp", tr.TestJobSpec.SubmissionID)
	tr.CreateTempLocations(t)
	tr.CopyFiles(t, "../examples/wrongans.cpp")

	warmCont := SharedTF.GetWarmContainer(t, ctx)

	// get events from RMQ
	interceptorQueue := make(chan amqp.Delivery, 100)
	routingKey := tr.TestJobSpec.SubmissionID
	if err := SharedTF.Rmqm.SubscribeToExchange(ctx, interceptorQueue, os.Getenv("DIRECT_EXCHANGE_NAME"), routingKey); err != nil {
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
			case <-ctx.Done():
				return
			case delivery, ok := <-interceptorQueue:
				if !ok {
					return
				}
				_ = delivery.Ack(false)

				var testEventStream utils.Event
				json.Unmarshal(delivery.Body, &testEventStream)

				if testEventStream.Type == "RESULT" {
					assert.String(t, "Wrong answer", testEventStream.Status)
					break // Type RESULT means stream has ended
				}

				counter++
				assert.String(t, "INFO", testEventStream.Type)
				assert.String(t, fmt.Sprintf("Wrong answer in test %v", counter), testEventStream.Status)
			}
		}
	}()

	// channel to safely receive the return value of ExecSubm
	type result struct {
		info utils.ContainerInfo
	}
	execChan := make(chan result, 1)

	go func() {
		info := warmCont.ExecSubm(ctx, tr.TestExecRules, tr.TestJobSpec, *SharedTF.Rmqm, *SharedTF.S3m)
		execChan <- result{info: info}
	}()

	// send execspec
	if err := json.NewEncoder(warmCont.Conn).Encode(&tr.TestAgentExecSpec); err != nil {
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

	t.Logf(`Container lifecycle data
			SubmissionID: %v
			Language: %v
			Version: %v
			Elapsed time: %vms
			Status: %v
			StatusInfo: %v
			Stderr: %v
			Stdout: %v`,
		contInfo.SubmissionId, contInfo.Language, contInfo.Version,
		contInfo.Interval, contInfo.Status, contInfo.StatusInfo, contInfo.ContainerStderr, contInfo.ContainerStdout)

}
