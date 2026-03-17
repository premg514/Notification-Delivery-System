package domain

import (
	"context"
	"notification-system/internal/domain/models"
	"time"
)

type NotificationRepository interface {
	CreateNotification(ctx context.Context, n models.Notification) error
	CreateNotificationWithDeliveries(ctx context.Context, n models.Notification, deliveries []models.Delivery) error
	GetNotificationByID(ctx context.Context, id string) (models.Notification, error)
	GetNotificationByIdempotencyKey(ctx context.Context, key string) (models.Notification, error)
	CreateNotificationIfAbsent(ctx context.Context, n models.Notification) (models.Notification, bool, error)
	UpdateDeliveryStatus(ctx context.Context, deliveryID string, status string) error
	UpdateDeliveryStatusWithRetry(ctx context.Context, deliveryID string, status string, retryCount int, deliveredAt *time.Time, lastError string) error
	CreateDeliveryRecord(ctx context.Context, d models.Delivery) error
	CreateDeliveriesIfAbsent(ctx context.Context, deliveries []models.Delivery) ([]models.Delivery, error)
}
