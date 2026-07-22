package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"local/testrunner/factory"
	"local/testrunner/repository"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dispatcher"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Test_Dispatcher_Subsystem_E2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tf := factory.NewTestFactory(t)
	tf.StartTestRMQ(t, ctx)
	tf.StartTestMinioS3(t, ctx)

	tr := repository.NewTestRepository(t)
	submissionPayload := tr.TestJobSpec

	if err := tf.S3m.UploadDirToS3(ctx, "ts001/v1", "../artifacts/ts001"); err != nil {
		t.Fatal(err)
	}

	interceptorQueue := make(chan amqp.Delivery, 5)

	err := tf.Rmqm.Subscribe(ctx, interceptorQueue, tf.RmqQueueName, "e2e-interceptor")
	if err != nil {
		t.Fatalf("E2E Setup Error: Failed to register test consumer: %v", err)
	}

	// MUST load configs or else test fails with nil
	dispatcher.LoadConfigs("../artifacts/config.example.yaml")

	// boot actual server mux inside a Live Test HTTP Container
	serverConfig := dispatcher.InitHTTPServer(ctx, tf.S3m, tf.Rmqm)
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
	_ = json.NewDecoder(resp.Body).Decode(&jsonResp)
	expectedJobID := jsonResp["job_id"]

	select {
	case delivery, ok := <-interceptorQueue:
		if !ok {
			t.Fatal("E2E Validation Failed: Interceptor channel closed abruptly")
		}

		// immediately clear out the message from the broker
		_ = delivery.Ack(false)

		// ASSERTION B: verify structural properties stayed unmutated inside the broker
		if delivery.MessageId != expectedJobID {
			t.Errorf("E2E Verification Failed: Message ID mismatch. Expected %s, got %s", expectedJobID, delivery.MessageId)
		}

		var brokerPayload dispatcher.SubmissionSpec
		_ = json.Unmarshal(delivery.Body, &brokerPayload)
		if brokerPayload.Language != "cpp" {
			t.Errorf("E2E Verification Failed: Data distortion detected! Language field got mutated into: %s", brokerPayload.Language)
		}

		t.Log("E2E Core Flow passed perfectly: HTTP -> Validation -> RMQ Wire.")
	case <-time.After(30 * time.Second):
		t.Fatal("E2E Validation Failed: Timeout reached before message broke into RabbitMQ")
	}
}
