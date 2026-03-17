package queue

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var retryDurations = []time.Duration{
	5 * time.Second,
	20 * time.Second,
	60 * time.Second,
}

type RabbitMQ struct {
	conn                     *amqp.Connection
	fanoutQueue              string
	deliveryQueue            string
	deliveryRetryQueuePrefix string
}

func New(url, fanoutQueue, deliveryQueue, deliveryRetryQueuePrefix string) (*RabbitMQ, error) {
	conn, err := amqp.DialConfig(url, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		return nil, err
	}

	rabbit := &RabbitMQ{
		conn:                     conn,
		fanoutQueue:              fanoutQueue,
		deliveryQueue:            deliveryQueue,
		deliveryRetryQueuePrefix: deliveryRetryQueuePrefix,
	}

	if err := rabbit.declareTopology(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return rabbit, nil
}

func (r *RabbitMQ) Connection() *amqp.Connection {
	return r.conn
}

func (r *RabbitMQ) Close() error {
	if r.conn == nil {
		return nil
	}
	return r.conn.Close()
}

func (r *RabbitMQ) FanoutQueueName() string {
	return r.fanoutQueue
}

func (r *RabbitMQ) DeliveryQueueName() string {
	return r.deliveryQueue
}

func (r *RabbitMQ) DeliveryRetryQueueName(retryNumber int) string {
	return fmt.Sprintf("%s.%d", r.deliveryRetryQueuePrefix, retryNumber)
}

func (r *RabbitMQ) declareTopology() error {
	channel, err := r.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	queueArgs := amqp.Table{
		"x-queue-type": "quorum",
	}

	if _, err = channel.QueueDeclare(r.fanoutQueue, true, false, false, false, queueArgs); err != nil {
		return err
	}

	if _, err = channel.QueueDeclare(r.deliveryQueue, true, false, false, false, queueArgs); err != nil {
		return err
	}

	for index, delay := range retryDurations {
		retryNumber := index + 1
		if _, err = channel.QueueDeclare(r.DeliveryRetryQueueName(retryNumber), true, false, false, false, amqp.Table{
			"x-queue-type":               "quorum",
			"x-message-ttl":              int(delay.Milliseconds()),
			"x-dead-letter-exchange":     "",
			"x-dead-letter-routing-key":  r.deliveryQueue,
		}); err != nil {
			return err
		}
	}

	return nil
}
