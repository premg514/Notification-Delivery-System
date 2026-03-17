package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Task[T any] struct {
	Envelope amqp.Delivery
	Job      T
}

type submitter[T any] interface {
	Submit(ctx context.Context, task Task[T]) error
}

type Consumer[T any] struct {
	channel   *amqp.Channel
	queueName string
	pool      submitter[T]
}

func NewConsumer[T any](rabbit *RabbitMQ, queueName string, prefetchCount int, pool submitter[T]) (*Consumer[T], error) {
	channel, err := rabbit.Connection().Channel()
	if err != nil {
		return nil, err
	}

	if err = channel.Qos(prefetchCount, 0, false); err != nil {
		_ = channel.Close()
		return nil, err
	}

	return &Consumer[T]{
		channel:   channel,
		queueName: queueName,
		pool:      pool,
	}, nil
}

func (c *Consumer[T]) Close() error {
	if c.channel == nil {
		return nil
	}
	return c.channel.Close()
}

func (c *Consumer[T]) Start(ctx context.Context) error {
	deliveries, err := c.channel.Consume(c.queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("rabbitmq deliveries channel closed unexpectedly for queue %s", c.queueName)
			}

			var job T
			if err := json.Unmarshal(delivery.Body, &job); err != nil {
				slog.Warn("invalid message payload rejected",
					"queue", c.queueName,
					"delivery_tag", delivery.DeliveryTag,
					"error", err,
				)
				_ = delivery.Reject(false)
				continue
			}

			if err := c.pool.Submit(ctx, Task[T]{Envelope: delivery, Job: job}); err != nil {
				if errors.Is(err, context.Canceled) {
					_ = delivery.Nack(false, true)
					return nil
				}
				_ = delivery.Nack(false, true)
				return err
			}
		}
	}
}
