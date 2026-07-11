package tests

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"local/runner/utils"
)

func TestProcessJobSpec_ValidJSON(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Microsecond)
	defer cancel()

	localQueue := make(chan amqp.Delivery, 1)

	localQueue <- amqp.Delivery{
		Body: []byte(`
			{
				"job_id": "job111", 
				"language": "c++", 
				"version": "c++17",
				"submission_id": "sub123",
				"filepath": "job111/sub123.cpp",
				"testset": "ts001",
				"testset_version": "v1"
			}
		`),
	}
	close(localQueue)

	var recievedJob utils.JobSpec
	callbackCalled := false

	err := utils.ProcessJobSpec(ctx, localQueue, func(job utils.JobSpec) {
		callbackCalled = true
		recievedJob = job
	})

	if err != nil {
		t.Fatalf("Expected no error from healthy loop, got: %v", err)
	}

	if !callbackCalled {
		t.Error("Expected onSuccess callback to be executed for valid JSON")
	}

	if recievedJob.JobId == "" {
		t.Errorf("Expected job spec field to be populated, but it was empty")
	}
}
