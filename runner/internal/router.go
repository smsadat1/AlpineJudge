package internal

import (
	"context"
	"encoding/json"
	"log"
	"shared"

	amqp "github.com/rabbitmq/amqp091-go"
)

// publish rmq specific event payload directly to RMQ in real time
func routeToRMQ(
	sockCtx context.Context, submissionID string, rmqm shared.RMQManager, exchangename string, payload []byte,
) {
	// unique routing key per submission using submissionID
	routingKey := submissionID

	if !json.Valid(payload) {
		log.Printf("Failed to stream event to RMQ: invalid JSON payload")
		return
	}
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        payload,
	}

	if err := rmqm.PublishToExchange(sockCtx, exchangename, routingKey, msg); err != nil {
		log.Printf("Failed to stream event to RMQ: %v", err)
	}
}
