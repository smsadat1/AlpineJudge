package integration

import (
	"context"
	"dispatcher"
	"encoding/json"
	"local/testrunner/factory"
	"local/testrunner/repository"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRMQPipeline_Integration(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tf := factory.NewTestFactory(t)
	tf.StartTestRMQ(t, ctx)

	tr := repository.NewTestRepository(t)
	submSpec := tr.TestSubmSpec

	// allocate a buffered channel for our consumer background routine to pipe into
	jobSpecsQueue := make(chan amqp.Delivery, 5)

	// 1. Turn on the consumer stream in the background
	go func() {
		if err := tf.Rmqm.Subscribe(ctx, jobSpecsQueue, tf.RmqQueueName, "test-rmq-consumer"); err != nil {
			t.Logf("Consumer registration failed or exited: %v\n", err)
		}
	}()

	// short sleep to ensure consumer registration completes on the RabbitMQ broker node
	time.Sleep(250 * time.Millisecond)

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
	if err := tf.Rmqm.Publish(ctx, tf.RmqQueueName, msg); err != nil {
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
		if receivedSpec.SubmissionID != "testsub001" || receivedSpec.Language != "cpp" {
			t.Errorf("Critical data distortion detected! Payload changed: %+v\n", receivedSpec)
		} else {
			t.Logf("Success! Jobspec field round-trip validation matched perfectly without mutation\n")
		}

	case <-time.After(3 * time.Second):
		t.Fatal("Timeout: Pipeline hung waiting for the consumer to grab the published message")
	}
}
