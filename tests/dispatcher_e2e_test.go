package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"dispatcher"
	"shared"
)

func Test_Dispatcher_Subsystem_E2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	t.Setenv("TEST_S3_URL", "http://localhost:9000")
	t.Setenv("TEST_S3_USERNAME", "minioadmin")
	t.Setenv("TEST_S3_PASSWORD", "minioadminpassword")
	t.Setenv("TEST_S3_BUCKET_NAME", "ajbucket-test-e2e-d")
	t.Setenv("TEST_S3_REGION_NAME", "us-east-1")
	t.Setenv("TEST_RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	t.Setenv("RABBITMQ_QUEUE_NAME", "job-queue-runner1")

	if err := dispatcher.LoadConfigs("artifacts/config.example.yaml"); err != nil {
		t.Fatal(err)
	}

	s3m, err := shared.InitS3Manager(
		ctx,
		os.Getenv("TEST_S3_BUCKET_NAME"),
		os.Getenv("TEST_S3_REGION_NAME"),
		os.Getenv("TEST_S3_USERNAME"),
		os.Getenv("TEST_S3_PASSWORD"),
		os.Getenv("TEST_S3_URL"),
	)

	if err != nil {
		t.Fatalf("E2E Setup Error: S3 failed: %v", err)
	}

	_, _ = s3m.CreateABucket(ctx, os.Getenv("TEST_S3_BUCKET_NAME"))

	if err := s3m.UploadDirToS3(ctx, "ts001/v1", "artifacts/ts001"); err != nil {
		t.Fatal(err)
	}

	rmqm, err := shared.NewRMQManager(ctx, os.Getenv("TEST_RABBITMQ_URL"))
	if err != nil {
		t.Fatalf("E2E Setup Error: RMQ failed: %v", err)
	}
	t.Cleanup(func() { rmqm.Close() })

	// a separate test interceptor queue to verify the message lands safely
	interceptorQueue := make(chan amqp.Delivery, 5)
	queueName := os.Getenv("RABBITMQ_QUEUE_NAME")

	err = rmqm.Subscribe(ctx, interceptorQueue, queueName, "e2e-interceptor")
	if err != nil {
		t.Fatalf("E2E Setup Error: Failed to register test consumer: %v", err)
	}

	// boot actual server mux inside a Live Test HTTP Container
	serverConfig := dispatcher.InitHTTPServer(ctx, s3m, rmqm)
	ts := httptest.NewServer(serverConfig.Handler)
	t.Cleanup(ts.Close)

	submissionPayload := dispatcher.SubmissionSpec{
		SubmissionID:   "s012",
		Language:       "cpp",
		Version:        "c++17",
		Source:         `#include<stdion.h>\nint main() \n{ std::cout << "Hello World\n";\n}`,
		Testset:        "ts001",
		TestsetVersion: "v1",
	}
	bodyBytes, _ := json.Marshal(submissionPayload)

	// fire actual HTTP client request over the wire
	resp, err := http.Post(ts.URL+"/submit", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("HTTP Transmit Error: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	// ASSERTION A: validate HTTP Layer response code
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status 202 Accepted, got: %d", resp.StatusCode)
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
