package utils

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

/*
* Continuously process data from localQueue
* Processed jobs are passed to onSuccess callback function
 */
func ProcessJobSpec(ctx context.Context, localQueue <-chan amqp.Delivery, onSuccess func(JobSpec)) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("Worker cotext cancelled. Exiting...")
			return nil
		case msg, ok := <-localQueue:
			if !ok {
				log.Println("Local queue cancelled. Exiting worker...")
				return nil
			}
			log.Printf("Worker processing job len: %v\n", len(msg.Body))

			var jobspec JobSpec
			err := json.Unmarshal(msg.Body, &jobspec)

			// NACK bad JSON and move on
			if err != nil {
				log.Printf("Error processsing job spec in JSON: %v | Raw: %v\n", err, jobspec)
				_ = msg.Nack(false, false)
				continue
			}

			log.Printf("Processed job spec: %v\n", jobspec)

			if msg.Acknowledger != nil {
				err = msg.Ack(false)
				if err != nil {
					log.Printf("Failed to ACK message: %v\n", err)
					return err
				}
			} else {
				log.Println("Skipping ACK logic (Running in Mock/Test environment)")
			}

			if onSuccess != nil {
				onSuccess(jobspec)
			}
		}
	}
}
