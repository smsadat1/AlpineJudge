package tests

import (
	"bytes"
	"context"
	"dispatcher/internal"
	"encoding/json"
	"io"
	"local/testrunner/factory"

	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Test_Dispatcher_Subsystem_E2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tf := factory.NewTestFactory(t)
	tf.StartTestRMQ(t, ctx)
	tf.StartTestMinioS3(t, ctx)

	tr := NewDispatcherRepository(t)
	submissionPayload := tr.TestJobSpec

	if err := tf.S3m.UploadDirToS3(ctx, "ts001/v1", "ts001"); err != nil {
		t.Fatal(err)
	}

	interceptorQueue := make(chan amqp.Delivery, 5)

	err := tf.Rmqm.Subscribe(ctx, interceptorQueue, tf.RmqQueueName, "e2e-interceptor")
	if err != nil {
		t.Fatalf("E2E Setup Error: Failed to register test consumer: %v", err)
	}

	// boot actual server mux inside a Live Test HTTP Container
	serverConfig := internal.InitHTTPServer(ctx, tf.S3m, tf.Rmqm)
	ts := httptest.NewServer(serverConfig.Handler)
	t.Cleanup(ts.Close)

	bodyBytes, _ := json.Marshal(submissionPayload)

	// fire actual HTTP client request over the wire
	resp, err := http.Post(ts.URL+"/submit", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("HTTP Transmit Error: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	// ASSERTION A: validate HTTP Layer response code
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status 202 Accepted, got: %d Details: %s", resp.StatusCode, string(body))
	}

	var jsonResp map[string]string
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		t.Fatalf("Failed to unmarshal response to JSON: %v", err)
	}
	expectedSubmID := jsonResp["submission_id"]

	select {
	case delivery, ok := <-interceptorQueue:
		if !ok {
			t.Fatal("E2E Validation Failed: Interceptor channel closed abruptly")
		}

		// immediately clear out the message from the broker
		_ = delivery.Ack(false)

		// ASSERTION B: verify structural properties stayed unmutated inside the broker
		if delivery.MessageId != expectedSubmID {
			t.Errorf("E2E Verification Failed: Message ID mismatch. Expected %s, got %s", expectedSubmID, delivery.MessageId)
		}

		var brokerPayload internal.SubmissionSpec
		_ = json.Unmarshal(delivery.Body, &brokerPayload)
		if brokerPayload.Language != "cpp" {
			t.Errorf("E2E Verification Failed: Data distortion detected! Language field got mutated into: %s", brokerPayload.Language)
		}

		t.Log("E2E Core Flow passed perfectly: HTTP -> Validation -> RMQ Wire.")
	case <-time.After(45 * time.Second):
		t.Fatal("E2E Validation Failed: Timeout reached before message broke into RabbitMQ")
	}
}
