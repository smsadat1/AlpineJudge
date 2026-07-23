package integration

import (
	"assert"
	"context"
	"encoding/json"
	"local/runner/executor"
	"local/testrunner/factory"
	"local/testrunner/repository"
	"log"
	"os"
	"testing"
	"time"
	"utils"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	amqp "github.com/rabbitmq/amqp091-go"
)

func Test_Execsubm_integration_test(t *testing.T) {

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

	image, err := client.GetImage(ctx, "ghcr.io/smsadat1/alpinejudge/gcc:test")
	if err != nil {
		t.Fatalf("Failed to find image in containerd: %v", err)
	}

	// cleanup stale socket & then prepare socket
	eventSocketPath := "../artifacts/agent.sock"
	_ = os.RemoveAll(eventSocketPath)

	tr := repository.NewTestRepository(t)
	tr.NewTestOCISpecOpts(t, tr.TestExecRules)

	containerID := "test-subm-" + time.Now().Format("150405")

	err, data := executor.Build_agentExecSpec(tr.TestExecRules)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("../artifacts/execspec1.json", data, os.ModeAppend); err != nil {
		t.Fatalf("Failed to create agent execspec json:  %v\n", err)
	}

	// configure OCI Spec (Env, Mounts, Process Args)
	container, err := client.NewContainer(
		ctx,
		containerID,
		containerd.WithNewSnapshot(containerID+"-snapshot", image),
		containerd.WithNewSpec(tr.TestOCISpecOpts...),
	)
	if err != nil {
		t.Fatalf("Failed to create containerd container: %v", err)
	}
	defer func() {
		// cleanup container with background context in case main ctx timed out
		_ = container.Delete(namespaces.WithNamespace(context.Background(), "test-namespace"), containerd.WithSnapshotCleanup)
	}()

	// sin up MinIO & RabbitMQ containers via Testcontainers
	tf := factory.NewTestFactory(t)
	tf.StartTestMinioS3(t, ctx)
	tf.StartTestRMQ(t, ctx)

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
				log.Printf("%v | %v", testEventStream.Status, testEventStream.Details)
				assert.String(t, testEventStream.EvenType, "ACCEPT")

				select {
				case collectMesg <- string(delivery.Body):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// run ExecSubm
	_ = executor.ExecSubm(ctx, container, tr.TestExecRules, tr.TestJobSpec, *tf.Rmqm, *tf.S3m)

	select {
	case msg, ok := <-collectMesg:
		if !ok {
			t.Errorf("Failed to capture live status event: %s", msg)
		}
		t.Logf("Successfully captured live status event: %s", msg)
	case <-ctx.Done():
		t.Fatal("Timed out waiting for live status message on RMQ")
	}

}
