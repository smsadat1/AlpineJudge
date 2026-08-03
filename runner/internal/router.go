package internal

import (
	"context"
	"encoding/json"
	"log"
	"shared"

	amqp "github.com/rabbitmq/amqp091-go"
)

// publish rmq specific event payload directly to RMQ in real time
func routeToRMQ(sockCtx context.Context, queuename string, rmqm shared.RMQManager, payload []byte) {

	if queuename == "" {
		log.Println("Failed to publish event: queue name is EMPTY")
		return
	}

	if !json.Valid(payload) {
		log.Printf("Failed to stream event to RMQ: invalid JSON payload")
		return
	}
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        payload,
	}

	if err := rmqm.Publish(sockCtx, queuename, msg); err != nil {
		log.Printf("Failed to stream event to RMQ: %v", err)
	}
}
