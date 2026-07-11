package tests

import (
	"context"
	"dispatcher"
	"encoding/json"
	"os"
	"shared"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRMQPipeline_Integration(t *testing.T) {

	t.Setenv("RABBITMQ_URL_TEST", "amqp://guest:guest@localhost:5672/")
	t.Setenv("RABBITMQ_QUEUE_NAME", "test-queue-1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rmqm, err := shared.NewRMQManager(ctx, os.Getenv("RABBITMQ_URL_TEST"))
	if err != nil {
		t.Fatal(err)
	}
	defer rmqm.Close()

	localQueue := make(chan amqp.Publishing, 5)
	jobSpecsQueue := make(chan amqp.Delivery, 5)

	go func() {
		if err := rmqm.Publish(ctx, os.Getenv("RABBITMQ_QUEUE_NAME"), localQueue); err != nil {
			t.Logf("Producer exited: %v\n", err)
		}
	}()

	go func() {
		if err := rmqm.Subscribe(ctx, jobSpecsQueue); err != nil {
			t.Logf("Consumer exited: %v\n", err)
		}
	}()

	// amqp.Dial backoff logic a microsecond to connect
	time.Sleep(250 * time.Millisecond)

	submSpec := dispatcher.SubmissionSpec{
		SubmissionID:   "s001",
		Language:       "cpp",
		Version:        "c++17",
		Source:         `#include<stdion.h>\nint main() \n{ std::cout << "Hello World\n";\n}`,
		Testset:        "ts001",
		TestsetVersion: "v1",
	}

	jsonedSpec, err := json.Marshal(submSpec)
	if err != nil {
		t.Fatal(err)
	}

	localQueue <- amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonedSpec,
	}

	select {
	case jobSpec := <-jobSpecsQueue:
		// auto acknowledge the message back to RabbitMQ to keep the queue clean
		_ = jobSpec.Ack(false)
		var receivedSpec dispatcher.SubmissionSpec

		if err := json.Unmarshal(jobSpec.Body, &receivedSpec); err != nil {
			t.Fatalf("Failed to decode received message: %v\n", err)
		}

		if receivedSpec.SubmissionID != "s001" || receivedSpec.Language != "cpp" {
			t.Errorf("Jobspec field mismatched! Got: %+v\n", receivedSpec)
		} else {
			t.Logf("Success! Jobspec field round-trip validation matched perfectly\n")
		}

	case <-time.After(2 * time.Second):
		t.Fatal("Pipeline timed out waiting for the consumer to receive the producer's message")
	}

	close(localQueue)
	close(jobSpecsQueue)
}
