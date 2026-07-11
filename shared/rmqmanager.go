package shared

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RMQManager struct {
	conn *amqp.Connection
	q    amqp.Queue
}

func NewRMQManager(ctx context.Context, amqpURL string) (*RMQManager, error) {

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

		if i < 6 {
			i++
		}
		conn, err = amqp.Dial(amqpURL)
	}

	log.Println("Connected to RabbitMQ server")

	// a temporary channel just to configure topology and declare the queue
	initCh, err := conn.Channel()
	failOnError(err, "Failed to open channel")
	defer initCh.Close()
	log.Println("Opened channel")

	maxQueueCap, _ := strconv.Atoi(os.Getenv("MAX_QUEUE_CAP"))
	err = initCh.Qos(
		maxQueueCap,
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS backpressure")

	queueName := os.Getenv("RABBITMQ_QUEUE_NAME")
	q, err := initCh.QueueDeclare(
		queueName,
		true,  // survive server restart
		false, // no auto delete
		false, // exclusive queue per runner service
		false, // wait
		nil,
	)
	failOnError(err, "Failed to declared queue")

	return &RMQManager{
		conn: conn,
		q:    q,
	}, nil
}

func (m *RMQManager) Subscribe(
	ctx context.Context, localQueue chan<- amqp.Delivery, consumerName string,
) error {

	// open a dedicated channel
	consumerCh, err := m.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open dedicated consumer channel: %w", err)
	}
	defer consumerCh.Close() // auto clean up channel

	// apply QoS prefetch backpressure limit directly to this channel
	if err := consumerCh.Qos(cap(localQueue), 0, false); err != nil {
		return fmt.Errorf("failed to set consumer QoS prefetch: %w", err)
	}

	msgs, err := consumerCh.Consume(
		m.q.Name,
		consumerName,
		false, // runner will send ACK later
		false, // exclusive
		true,  // no local
		false, // no wait
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
	ctx context.Context, localQueue <-chan amqp.Publishing,
) error {

	// a completely isolated channel dedicated to publishing tasks
	prodCh, err := m.conn.Channel()
	if err != nil {
		return fmt.Errorf("Failed to open dedicated producer channel: %w\n", err)
	}
	defer prodCh.Close()

	log.Println("Producer initialized. Ready to transmit payloads...")

	// continuous loop to drain the channel safely without data loss
	for msg := range localQueue {
		// short 5-second timeout context strictly for this specific publish

		pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
		err := prodCh.PublishWithContext(pubCtx, "", m.q.Name, false, false, msg)
		pubCancel() // clean up context instantly inside the loop

		if err != nil {
			log.Printf("Failed to publish message: %v\n", err)
			continue // continue for later messages
		}

		log.Printf("Sent message successfully! Type: %s, Body length: %d\n", msg.ContentType, len(msg.Body))
	}
	return nil
}

// close gracefully terminates the root network connection handle
func (m *RMQManager) Close() {
	if m.conn != nil {
		m.conn.Close()
		log.Println("RabbitMQ server connection closed safely")
	}
}
