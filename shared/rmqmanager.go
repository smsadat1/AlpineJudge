package shared

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RMQManager struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	q    amqp.Queue
}

func NewRMQManager(ctx context.Context) (*RMQManager, error) {

	err := godotenv.Load()
	if err != nil {
		// fallback: If running from repo_root/, look explicitly inside /runner/.env
		_ = godotenv.Load(filepath.Join("runner", ".env"))
	}

	amqpURL := os.Getenv("RABBITMQ_URL_DEV")
	if amqpURL == "" {
		return nil, fmt.Errorf("RMQ url not found in environment!\n")
	}

	log.Printf("Connecting to RabbitMQ server at %s\n", amqpURL)

	conn, err := amqp.Dial(amqpURL)
	// connection retry (exponential backoff | 10s, 20s, 30s, 40s, 50s, 60s, 60s, 60s ...)
	i := 1
	for err != nil {
		log.Printf("Failed to connect to RabbitMQ server. Retrying in %vs ...\n", 10*i)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("Exiting RMQ Client service...\n")
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

	return &RMQManager{
		conn: conn,
		ch:   ch,
		q:    q,
	}, nil
}

func (m *RMQManager) Subscribe(
	ctx context.Context, localQueue chan<- amqp.Delivery,
) error {
	msgs, err := m.ch.Consume(
		m.q.Name,
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
			return fmt.Errorf("Stopping consumer loop...\n")
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("RabbitMQ channel closed unexpectedly\n")
			}

			//  naturally BLOCK here if localQueue reaches MAX_QUEUE_CAP.
			localQueue <- msg
		}
	}
}

func (m *RMQManager) Publish(
	ctx context.Context, queueName string, localQueue <-chan amqp.Publishing,
) error {
	targetQueue, err := m.ch.QueueDeclare(
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
		pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
		err = m.ch.PublishWithContext(pubCtx, "", targetQueue.Name, false, false, msg)
		pubCancel() // clean up context instantly inside the loop

		if err != nil {
			log.Printf("Failed to publish message: %v\n", err)
			continue // continue for later messages
		}

		log.Printf("Sent message successfully! Type: %s, Body length: %d\n", msg.ContentType, len(msg.Body))
	}
	return nil
}

func (m *RMQManager) Close() {
	if m.ch != nil {
		m.ch.Close()
		log.Println("RabbitMQ channel closed safely")
	}
	if m.conn != nil {
		m.conn.Close()
		log.Println("RabbitMQ server connection closed safely")
	}
}
