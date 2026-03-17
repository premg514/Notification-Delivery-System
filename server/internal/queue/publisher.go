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
	channel       *amqp.Channel
	rabbit        *RabbitMQ
	confirmations chan amqp.Confirmation
	mu            sync.Mutex
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

	confirmations := channel.NotifyPublish(make(chan amqp.Confirmation, 1024))

	return &Publisher{
		channel:       channel,
		rabbit:        rabbit,
		confirmations: confirmations,
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

	if err := p.channel.PublishWithContext(ctx, "", queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		return err
	}

	select {
	case confirmation := <-p.confirmations:
		if !confirmation.Ack {
			return fmt.Errorf("rabbitmq publish was not acknowledged for queue %s", queueName)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
