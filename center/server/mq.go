package main

import (
	"fmt"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const jobQueueName = "swarm_jobs"

// MQ wraps a RabbitMQ connection for publishing and consuming job IDs.
type MQ struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// ConnectMQ connects to RabbitMQ with retries.
func ConnectMQ() (*MQ, error) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@rabbitmq:5672/"
	}

	var conn *amqp.Connection
	var err error

	// Retry connection up to 30 times (RabbitMQ may start slowly)
	for i := 0; i < 30; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		logWarn(fmt.Sprintf("rabbitmq not ready (attempt %d/30): %v", i+1, err))
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq after 30 attempts: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the job queue (durable so jobs survive broker restarts)
	_, err = ch.QueueDeclare(
		jobQueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Fair dispatch: prefetch 1 job at a time per consumer
	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	logInfo("connected to rabbitmq")
	return &MQ{conn: conn, ch: ch}, nil
}

// Publish sends a job ID to the queue.
func (mq *MQ) Publish(jobID string) error {
	return mq.ch.Publish(
		"",           // default exchange
		jobQueueName, // routing key = queue name
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // survive broker restart
			ContentType:  "text/plain",
			Body:         []byte(jobID),
		},
	)
}

// Consume returns a channel of job IDs from the queue.
func (mq *MQ) Consume() (<-chan amqp.Delivery, error) {
	return mq.ch.Consume(
		jobQueueName,
		"",    // consumer tag (auto-generated)
		false, // auto-ack (we ack manually after processing)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
}

// Close shuts down the connection.
func (mq *MQ) Close() {
	if mq.ch != nil {
		mq.ch.Close()
	}
	if mq.conn != nil {
		mq.conn.Close()
	}
}
