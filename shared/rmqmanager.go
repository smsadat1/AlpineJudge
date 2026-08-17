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

func (m *RMQManager) SubscribeToExchange(
	ctx context.Context,
	localQueue chan<- amqp.Delivery,
	exchangeName string,
	routingKey string,
) error {
	// open a dedicated channel
	consumerCh, err := m.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open dedicated consumer channel: %w", err)
	}

	if err := consumerCh.ExchangeDeclare(
		exchangeName,
		"direct", // kind
		true,     // durable
		false,    // autoDelete
		false,    // internal
		false,    // noWait
		nil,      // args
	); err != nil {
		consumerCh.Close()
		return fmt.Errorf("failed to declare exchange %s: %w", exchangeName, err)
	}

	// Passing "" as queue name so RabbitMQ generate a unique server-side name on it's own
	q, err := consumerCh.QueueDeclare(
		"",    // empty string = auto-generated unique name (e.g., amq.gen-J38...)
		false, // durable
		true,  // autoDelete (deleted when consumer disconnects)
		true,  // exclusive (only this channel can access)
		false, // noWait
		nil,   // args
	)
	if err != nil {
		consumerCh.Close()
		return fmt.Errorf("failed to declare transient queue: %w", err)
	}

	// bind queue to exchange using the submission's routing key
	if err := consumerCh.QueueBind(
		q.Name,
		routingKey,
		exchangeName,
		false, // noWait
		nil,   // args
	); err != nil {
		consumerCh.Close()
		return fmt.Errorf("failed to bind queue %s to key %s: %w", q.Name, routingKey, err)
	}

	// apply QoS backpressure limit matching channel capacity
	if err := consumerCh.Qos(cap(localQueue), 0, false); err != nil {
		consumerCh.Close()
		return fmt.Errorf("failed to set consumer QoS: %w", err)
	}

	// register consumer
	consumerTag := fmt.Sprintf("sse_%s_%d", routingKey, time.Now().UnixNano())
	msgs, err := consumerCh.Consume(
		q.Name,
		consumerTag,
		false, // autoAck = true (transient SSE log streaming)
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		consumerCh.Close()
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("[RMQ] Subscribed to exchange '%s' [Key: %s] via temp queue '%s'", exchangeName, routingKey, q.Name)

	// pipe messages to local Go channel & manage cleanup
	go func() {
		// Closing consumerCh automatically deletes the exclusive/auto-delete queue in RMQ
		defer consumerCh.Close()

		for {
			select {
			case <-ctx.Done():
				log.Printf("[RMQ] Client context cancelled. Tearing down stream for key: %s", routingKey)
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
	_, err := m.pubCh.QueueDeclare(queueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // exclusive
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to declare target publish queue: %w", err)
	}

	err = m.pubCh.PublishWithContext(ctx, "", queueName, false, false, msg)

	if err != nil {
		return fmt.Errorf("Failed to publish message: %v\n", err)
	}
	return nil
}

func (m *RMQManager) PublishToExchange(
	ctx context.Context, exchangename string, routingKey string, msg amqp.Publishing,
) error {

	if err := m.pubCh.ExchangeDeclare(
		exchangename,
		"direct", // kind
		true,     // durable
		false,    // autoDelete
		false,    // internal
		false,    // noWait
		nil,      // args
	); err != nil {
		return fmt.Errorf("failed to declare target exchnage: %w", err)
	}

	if err := m.pubCh.PublishWithContext(ctx,
		exchangename,
		routingKey, // key
		false,      // mandatory
		false,      // immediate (true is deprecated, must be false)
		msg,
	); err != nil {
		return fmt.Errorf("Failed to publish message on exchange (%v): %v\n", exchangename, err)
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
