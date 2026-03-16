package domain

import (
	"context"
	"notification-system/internal/domain/models"
)

type NotificationRepository interface {
	CreateNotification(ctx context.Context, n models.Notification) error
	GetNotificationByID(ctx context.Context, id string) (models.Notification, error)
	UpdateDeliveryStatus(ctx context.Context, deliveryID string, status string) error
	CreateDeliveryRecord(ctx context.Context, d models.Delivery) error
}