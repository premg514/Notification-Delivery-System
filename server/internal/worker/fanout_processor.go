package worker

import (
	"context"
	"notification-system/internal/domain"
	"notification-system/internal/domain/models"
	"notification-system/internal/service"
	"time"
)

type DeliveryBatchPublisher interface {
	PublishDeliveryBatch(ctx context.Context, job models.DeliveryBatchJob) error
}

type FanoutProcessor struct {
	repo              domain.NotificationRepository
	publisher         DeliveryBatchPublisher
	fanoutBatchSize   int
	deliveryBatchSize int
}

func NewFanoutProcessor(repo domain.NotificationRepository, publisher DeliveryBatchPublisher, fanoutBatchSize, deliveryBatchSize int) *FanoutProcessor {
	if fanoutBatchSize <= 0 {
		fanoutBatchSize = 5000
	}
	if deliveryBatchSize <= 0 {
		deliveryBatchSize = 500
	}

	return &FanoutProcessor{
		repo:              repo,
		publisher:         publisher,
		fanoutBatchSize:   fanoutBatchSize,
		deliveryBatchSize: deliveryBatchSize,
	}
}

func (p *FanoutProcessor) Process(ctx context.Context, job models.FanoutJob) error {
	now := time.Now().UTC()
	for start := 0; start < len(job.TargetUserIDs); start += p.fanoutBatchSize {
		end := min(start+p.fanoutBatchSize, len(job.TargetUserIDs))
		deliveries := make([]models.Delivery, 0, end-start)

		for _, userID := range job.TargetUserIDs[start:end] {
			deliveries = append(deliveries, models.Delivery{
				ID:             service.NewID(),
				UserID:         userID,
				NotificationID: job.NotificationID,
				Status:         "pending",
				RetryCount:     0,
				UpdatedAt:      now,
			})
		}

		inserted, err := p.repo.CreateDeliveriesIfAbsent(ctx, deliveries)
		if err != nil {
			return err
		}

		for batchStart := 0; batchStart < len(inserted); batchStart += p.deliveryBatchSize {
			batchEnd := min(batchStart+p.deliveryBatchSize, len(inserted))
			items := make([]models.DeliveryAttempt, 0, batchEnd-batchStart)
			for _, delivery := range inserted[batchStart:batchEnd] {
				items = append(items, models.DeliveryAttempt{
					DeliveryID:     delivery.ID,
					UserID:         delivery.UserID,
					Title:          job.Title,
					Message:        job.Message,
					Priority:       job.Priority,
					RetryCount:     0,
					IdempotencyKey: job.IdempotencyKey,
				})
			}

			if len(items) == 0 {
				continue
			}

			if err := p.publisher.PublishDeliveryBatch(ctx, models.DeliveryBatchJob{
				NotificationID: job.NotificationID,
				Items:          items,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
