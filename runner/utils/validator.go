package utils

import (
	"context"
	"encoding/json"
	"log"
	"shared"

	amqp "github.com/rabbitmq/amqp091-go"
)

func ProcessJobSpec(
	ctx context.Context, msg amqp.Delivery, ssequeue string) (shared.JobSpec, error) {

	log.Printf("Worker processing job len: %v\n", len(msg.Body))

	var jobspec shared.JobSpec
	err := json.Unmarshal(msg.Body, &jobspec)

	// NACK bad JSON and move on
	if err != nil {
		log.Printf("Error processsing job spec in JSON: %v | Raw: %v\n", err, jobspec)
		_ = msg.Nack(false, false)
		return shared.JobSpec{}, err
	}

	// log.Printf("Processed job spec: %v\n", jobspec)
	return jobspec, nil
}
