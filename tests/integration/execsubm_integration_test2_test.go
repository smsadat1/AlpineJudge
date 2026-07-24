package integration

import (
	"assert"
	"context"
	"encoding/json"
	"local/runner/executor"
	"local/testrunner/factory"
	"local/testrunner/repository"
	"os"
	"testing"
	"time"
	"utils"

	"github.com/containerd/containerd/v2/pkg/namespaces"
	amqp "github.com/rabbitmq/amqp091-go"
)

func Test_Execsubm_integration_test_TLE(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		namespaces.WithNamespace(context.Background(), "test-namespace"),
		45*time.Second,
	)
	defer cancel()

	// cleanup stale socket & then prepare socket
	eventSocketPath := "../artifacts/agent.sock"
	_ = os.RemoveAll(eventSocketPath)

	tr := repository.NewTestRepository(t)
	tr.TestExecRules.CodePathHost = "../artifacts/timekiller.cpp"
	tr.NewTestOCISpecOpts(t, tr.TestExecRules)

	err, data := executor.Build_agentExecSpec(tr.TestExecRules)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("../artifacts/execspec1.json", data, os.ModeAppend); err != nil {
		t.Fatalf("Failed to create agent execspec json:  %v\n", err)
	}

	// spin up MinIO & RabbitMQ containers via Testcontainers
	tf := factory.NewTestFactory(t)
	tf.StartTestMinioS3(t, ctx)
	tf.StartTestRMQ(t, ctx)
	tf.StartRawContainer(t, ctx, *tr)

	// get events from RMQ
	interceptorQueue := make(chan amqp.Delivery, 100)
	collectMesg := make(chan string, 100)

	if err := tf.Rmqm.Subscribe(ctx, interceptorQueue, tr.TestExecRules.EventQueueName, "test-consoomer"); err != nil {
		t.Fatalf("Failed to subscribe to queue: %v", err)
	}

	// Goroutine reading events cleanly with context cancellation
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-interceptorQueue:
				if !ok {
					return
				}
				_ = delivery.Ack(false)

				var testEventStream utils.AgentEventSpec
				json.Unmarshal(delivery.Body, &testEventStream)

				// no event stream happends for this case
				// log.Printf("%v", testEventStream)

				select {
				case collectMesg <- string(delivery.Body):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// run ExecSubm
	result := executor.ExecSubm(ctx, tf.RawContainer, tr.TestExecRules, tr.TestJobSpec, *tf.Rmqm, *tf.S3m)

	select {
	case msg, ok := <-collectMesg:
		if !ok {
			t.Errorf("Failed to capture live status event: %s", msg)
		}
	case <-ctx.Done():
		// rabiitmq will timeout for TLE case so no point of making it fatal
		// t.Fatal("Timed out waiting for live status message on RMQ")
	}

	assert.String(t, string(utils.VerdictTLE), string(result.Status))
}
