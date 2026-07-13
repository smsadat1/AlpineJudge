package shared

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RMQManager struct {
	conn  *amqp.Connection
	pubCh *amqp.Channel
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s %s\n", err, msg)
	}
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

	// Initialize the global Publisher channel
	pubCh, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open global pub channel: %w", err)
	}

	return &RMQManager{
		conn:  conn,
		pubCh: pubCh,
	}, nil
}

func (m *RMQManager) Subscribe(
	ctx context.Context, localQueue chan<- amqp.Delivery, queueName string, consumerTag string,
) error {

	// open a dedicated channel
	consumerCh, err := m.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open dedicated consumer channel: %w", err)
	}

	// Declare the target queue (ensures it exists before consuming)
	q, err := consumerCh.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		consumerCh.Close()
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	// apply QoS prefetch backpressure limit directly to this channel
	if err := consumerCh.Qos(cap(localQueue), 0, false); err != nil {
		return fmt.Errorf("failed to set consumer QoS prefetch: %w", err)
	}

	msgs, err := consumerCh.Consume(
		q.Name,
		consumerTag,
		false, // runner will send ACK later
		false, // exclusive
		true,  // no local
		false, // no wait
		nil,   // args
	)
	failOnError(err, "Failed to register consumer")
	log.Println("Consumer registered. Piping data to Go channel")

	// Pipe the data frames in a background worker
	go func() {
		// clean up the transient channel the moment the context dies
		// (example: when the HTTP SSE client closes their browser tab)
		defer consumerCh.Close()
		for {
			select {
			case <-ctx.Done():
				log.Printf("Closing subscription stream for tag: %s", consumerTag)
				return
			case d, ok := <-msgs:
				if !ok {
					return
				}
				localQueue <- d
			}
		}
	}()

	return nil
}

func (m *RMQManager) Publish(ctx context.Context, queueName string, msg amqp.Publishing) error {

	// Ensure target queue exists before pushing
	_, err := m.pubCh.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare target publish queue: %w", err)
	}

	err = m.pubCh.PublishWithContext(ctx, "", queueName, false, false, msg)

	if err != nil {
		return fmt.Errorf("Failed to publish message: %v\n", err)
	}
	return nil
}

// close gracefully terminates the root network connection handle
func (m *RMQManager) Close() {
	if m.pubCh != nil {
		_ = m.pubCh.Close()
	}
	if m.conn != nil {
		_ = m.conn.Close()
	}
}
