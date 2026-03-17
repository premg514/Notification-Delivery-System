package worker

import (
	"context"
	"errors"
	"log/slog"
	"notification-system/internal/domain"
	"notification-system/internal/domain/models"
	"notification-system/internal/observability"
	"time"
)

type RetryBatchPublisher interface {
	PublishDeliveryRetry(ctx context.Context, retryNumber int, job models.DeliveryBatchJob) error
}

type DeliverySender interface {
	Send(ctx context.Context, job models.DeliveryAttempt) error
}

type DeliveryProcessor struct {
	repo             domain.NotificationRepository
	publisher        RetryBatchPublisher
	sender           DeliverySender
	maxRetries       int
	attemptsPerBatch int
}

func NewDeliveryProcessor(repo domain.NotificationRepository, publisher RetryBatchPublisher, sender DeliverySender, maxRetries, attemptsPerBatch int) *DeliveryProcessor {
	if attemptsPerBatch <= 0 {
		attemptsPerBatch = 32
	}

	return &DeliveryProcessor{
		repo:             repo,
		publisher:        publisher,
		sender:           sender,
		maxRetries:       maxRetries,
		attemptsPerBatch: attemptsPerBatch,
	}
}

func (p *DeliveryProcessor) Process(ctx context.Context, job models.DeliveryBatchJob) error {
	retryItems := make([]models.DeliveryAttempt, 0)

	for start := 0; start < len(job.Items); start += p.attemptsPerBatch {
		end := min(start+p.attemptsPerBatch, len(job.Items))
		for _, item := range job.Items[start:end] {
			if err := p.sender.Send(ctx, item); err == nil {
				deliveredAt := time.Now().UTC()
				if updateErr := p.repo.UpdateDeliveryStatusWithRetry(ctx, item.DeliveryID, "sent", item.RetryCount, &deliveredAt, ""); updateErr != nil {
					return updateErr
				}
				continue
			} else {
				nextRetry := item.RetryCount + 1
				lastError := err.Error()
				if nextRetry > p.maxRetries {
					if updateErr := p.repo.UpdateDeliveryStatusWithRetry(ctx, item.DeliveryID, "failed", item.RetryCount, nil, lastError); updateErr != nil {
						return updateErr
					}
					continue
				}

				item.RetryCount = nextRetry
				retryItems = append(retryItems, item)
				if updateErr := p.repo.UpdateDeliveryStatusWithRetry(ctx, item.DeliveryID, "retrying", nextRetry, nil, lastError); updateErr != nil {
					return updateErr
				}
			}
		}
	}

	if len(retryItems) == 0 {
		return nil
	}

	retryNumber := retryItems[0].RetryCount
	if err := p.publisher.PublishDeliveryRetry(ctx, retryNumber, models.DeliveryBatchJob{
		NotificationID: job.NotificationID,
		Items:          retryItems,
	}); err != nil {
		return err
	}
	observability.AddDeliveryRetryScheduled(len(retryItems))

	slog.Info("delivery retries scheduled", "notification_id", job.NotificationID, "retry_count", len(retryItems), "attempt", retryNumber)
	return nil
}

type LoggingSender struct{}

func (LoggingSender) Send(ctx context.Context, job models.DeliveryAttempt) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(20 * time.Millisecond):
		if job.UserID == "" {
			return errors.New("missing user id")
		}
		slog.Info("delivery sent", "delivery_id", job.DeliveryID, "user_id", job.UserID, "priority", job.Priority, "retry_count", job.RetryCount)
		return nil
	}
}
