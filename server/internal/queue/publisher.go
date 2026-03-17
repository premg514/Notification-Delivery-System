package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"notification-system/internal/domain/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	channel *amqp.Channel
	rabbit  *RabbitMQ
	mu      sync.Mutex
}

func NewPublisher(rabbit *RabbitMQ) (*Publisher, error) {
	channel, err := rabbit.Connection().Channel()
	if err != nil {
		return nil, err
	}

	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		return nil, err
	}

	return &Publisher{
		channel: channel,
		rabbit:  rabbit,
	}, nil
}

func (p *Publisher) Close() error {
	if p.channel == nil {
		return nil
	}
	return p.channel.Close()
}

func (p *Publisher) PublishFanout(ctx context.Context, job models.FanoutJob) error {
	return p.publishToQueue(ctx, p.rabbit.FanoutQueueName(), job)
}

func (p *Publisher) PublishDeliveryBatch(ctx context.Context, job models.DeliveryBatchJob) error {
	return p.publishToQueue(ctx, p.rabbit.DeliveryQueueName(), job)
}

func (p *Publisher) PublishDeliveryRetry(ctx context.Context, retryNumber int, job models.DeliveryBatchJob) error {
	return p.publishToQueue(ctx, p.rabbit.DeliveryRetryQueueName(retryNumber), job)
}

func (p *Publisher) publishToQueue(ctx context.Context, queueName string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	confirmations := p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	if err := p.channel.PublishWithContext(ctx, "", queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return err
	}

	select {
	case confirmation := <-confirmations:
		if !confirmation.Ack {
			return fmt.Errorf("rabbitmq publish was not acknowledged for queue %s", queueName)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
