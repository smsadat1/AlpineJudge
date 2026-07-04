package shared

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

func RMQProducer(ctx context.Context, localQueue <-chan amqp.Publishing) {
	err := godotenv.Load()
	if err != nil {
		// fallback: If running from repo_root/, look explicitly inside /runner/.env
		_ = godotenv.Load(filepath.Join("runner", ".env"))
	}

	amqpURL := os.Getenv("RABBITMQ_URL_DEV")
	if amqpURL == "" {
		log.Println("RMQ url not found in environment!")
		return
	}

	log.Printf("Connecting to RabbitMQ server at %s\n", amqpURL)

	conn, err := amqp.Dial(amqpURL)
	// connection retry (exponential backoff | 10s, 20s, 30s, 40s, 50s, 60s, 60s, 60s ...)
	i := 1
	for err != nil {
		log.Printf("Failed to connect to RabbitMQ server. Retrying in %vs ...\n", 10*i)

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(10*i) * time.Second):
		}

		if i <= 6 {
			i++
		} else {
			i = 6
		}
		conn, err = amqp.Dial(amqpURL)
	}
	defer conn.Close()

	log.Println("Connected to RabbitMQ server")

	ch, err := conn.Channel()
	failOnError(err, "Failed to open channel")
	defer ch.Close()
	log.Println("Opened channel")

	queueName := os.Getenv("RABBITMQ_QUEUE_NAME")
	q, err := ch.QueueDeclare(
		queueName,
		true,  // survive server restart
		false, // no auto delete
		false, // exclusive queue per runner service
		true,  // wait
		nil,
	)
	failOnError(err, "Failed to declared queue")

	log.Println("Producer initialized. Ready to transmit payloads...")

	// continuous loop to drain the channel safely without data loss
	for msg := range localQueue {
		// short 5-second timeout context strictly for this specific publish
		_, pubCancel := context.WithTimeout(ctx, 5*time.Second)
		err = ch.PublishWithContext(ctx, "", q.Name, false, false, <-localQueue)
		pubCancel() // clean up context instantly inside the loop

		if err != nil {
			log.Printf("Failed to publish message: %v\n", err)
			continue // continue for later messages
		}

		log.Printf("Sent message successfully! Type: %s, Body length: %d\n", msg.ContentType, len(msg.Body))
	}
	log.Println("Local queue channel closed. Exiting producer routine gracefully")
}
