package tests

import (
	"context"
	"testing"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"

	"local/runner/utils"
	"shared"
)

func TestLiveRabbitMQIntegration(t *testing.T) {

	_ = godotenv.Load("../../runner/.env")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var recievedJob utils.JobSpec
	callbackCalled := false
	localQueue := make(chan amqp.Delivery, 10)
	done := make(chan struct{})

	// start bg process to wait for data
	go func() {
		_ = utils.ProcessJobSpec(ctx, localQueue, func(job utils.JobSpec) {
			callbackCalled = true
			recievedJob = job
			close(done)
		})
	}()

	shared.RMQConsumer(ctx, localQueue)

	select {
	case <-done:
		t.Log("Callback executed")
	case <-ctx.Done():
		t.Log("Context finished or timed out")
	}

	if !callbackCalled {
		t.Error("Expected onSuccess callback to be executed for valid JSON")
	}

	if recievedJob.JobId == "" {
		t.Errorf("Expected job spec field to be populated, but it was empty")
	}

}
