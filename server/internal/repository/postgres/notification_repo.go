package postgres

import (
	"context"
	"notification-system/internal/domain/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) CreateNotification(ctx context.Context, n models.Notification) error {
	return nil
}

func (r *NotificationRepository) GetNotificationByID(ctx context.Context, id string) (models.Notification, error) {
	var notification models.Notification
	return notification, nil
}

func (r *NotificationRepository) UpdateDeliveryStatus(ctx context.Context, deliveryID string, status string) error {
	return nil
}

func (r *NotificationRepository) CreateDeliveryRecord(ctx context.Context, d models.Delivery) error {
	return nil
}