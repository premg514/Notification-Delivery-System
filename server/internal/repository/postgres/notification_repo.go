package postgres

import (
	"context"
	"notification-system/internal/domain/models"
)

type NotificationRepository struct {
	// later we will add DB connection here
}

func (r *NotificationRepository) CreateNotification(ctx context.Context, notification models.Notification) error {
	// SQL query will come here later
	return nil
}

func (r *NotificationRepository) GetNotificationByID(ctx context.Context, id string) (models.Notification, error) {
	var notification models.Notification
	return notification, nil
}

func (r *NotificationRepository) UpdateDeliveryStatus(ctx context.Context, id string, status string) error {
	return nil
}

func (r *NotificationRepository) CreateDeliveryRecord(ctx context.Context, delivery models.Delivery) error {
	return nil
}