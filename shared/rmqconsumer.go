package shared

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

func RMQConsumer(ctx context.Context, localQueue chan<- amqp.Delivery) {

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

	maxQueueCap, _ := strconv.Atoi(os.Getenv("MAX_QUEUE_CAP"))
	err = ch.Qos(
		maxQueueCap,
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS backpressure")

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

	msgs, err := ch.Consume(
		q.Name,
		"runner_consumer",
		false, // runner will send ACK later
		false, // exclusive
		true,  // no local
		true,  // no wait
		nil,   // args
	)
	failOnError(err, "Failed to register consumer")
	log.Println("Consumer registered. Piping data to Go channel")

	// pull from RMQ chan and pass to localQueue
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping consumer loop...")
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Println("RabbitMQ channel closed unexpectedly")
				return
			}

			//  naturally BLOCK here if localQueue reaches MAX_QUEUE_CAP.
			localQueue <- msg
		}
	}
}
