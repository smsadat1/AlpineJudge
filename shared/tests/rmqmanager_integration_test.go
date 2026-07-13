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

	// allocate a buffered channel for our consumer background routine to pipe into
	jobSpecsQueue := make(chan amqp.Delivery, 5)

	queueName := os.Getenv("RABBITMQ_QUEUE_NAME")

	// 1. Turn on the consumer stream in the background
	go func() {
		if err := rmqm.Subscribe(ctx, jobSpecsQueue, queueName, "test-rmq-consumer"); err != nil {
			t.Logf("Consumer registration failed or exited: %v\n", err)
		}
	}()

	// short sleep to ensure consumer registration completes on the RabbitMQ broker node
	time.Sleep(250 * time.Millisecond)

	// 2. Create test spec payload
	submSpec := dispatcher.SubmissionSpec{
		SubmissionID:   "s001",
		Language:       "cpp",
		Version:        "c++17",
		Source:         `#include<iostream>\nint main() \n{ std::cout << "Hello World\n";\n}`,
		Testset:        "ts001",
		TestsetVersion: "v1",
	}

	jsonedSpec, err := json.Marshal(submSpec)
	if err != nil {
		t.Fatal(err)
	}

	msg := amqp.Publishing{
		ContentType: "application/json",
		MessageId:   submSpec.SubmissionID,
		Body:        jsonedSpec,
	}

	// 3. Transmit the payload into RMQ
	if err := rmqm.Publish(ctx, queueName, msg); err != nil {
		t.Fatalf("Failed to execute producer execution line: %v", err)
	}
	t.Log("Message successfully handed off to the RabbitMQ broker exchange.")

	// 4. Capture the message as it flows down out of the background subscription pipeline
	select {
	case jobSpec, ok := <-jobSpecsQueue:
		if !ok {
			t.Fatal("Failed to read from pipeline: consumer channel closed early")
		}

		// Send manual ACK acknowledgement back to the broker to keep the test environment clean
		_ = jobSpec.Ack(false)

		var receivedSpec dispatcher.SubmissionSpec
		if err := json.Unmarshal(jobSpec.Body, &receivedSpec); err != nil {
			t.Fatalf("Failed to decode received message wire bytes: %v\n", err)
		}

		// 5. Perform payload round-trip validation assertions
		if receivedSpec.SubmissionID != "s001" || receivedSpec.Language != "cpp" {
			t.Errorf("Critical data distortion detected! Payload changed: %+v\n", receivedSpec)
		} else {
			t.Logf("Success! Jobspec field round-trip validation matched perfectly without mutation\n")
		}

	case <-time.After(3 * time.Second):
		t.Fatal("Timeout: Pipeline hung waiting for the consumer to grab the published message")
	}
}
