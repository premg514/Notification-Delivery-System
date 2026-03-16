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

// Create a notification
func (r *NotificationRepository) CreateNotification(ctx context.Context, n models.Notification) error {

	query := `
	INSERT INTO notifications (id, title, message)
	VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(ctx, query, n.ID, n.Title, n.Message)

	if err != nil {
		return err
	}

	return nil
}

// Fetch notification by ID
func (r *NotificationRepository) GetNotificationByID(ctx context.Context, id string) (models.Notification, error) {

	query := `
	SELECT id, title, message, created_at
	FROM notifications
	WHERE id = $1
	`

	var notification models.Notification

	err := r.db.QueryRow(ctx, query, id).Scan(
		&notification.ID,
		&notification.Title,
		&notification.Message,
		&notification.CreatedAt,
	)

	if err != nil {
		return models.Notification{}, err
	}

	return notification, nil
}

// Update delivery status
func (r *NotificationRepository) UpdateDeliveryStatus(ctx context.Context, deliveryID string, status string) error {

	query := `
	UPDATE deliveries
	SET status = $1
	WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, status, deliveryID)

	if err != nil {
		return err
	}

	return nil
}

// Create delivery record
func (r *NotificationRepository) CreateDeliveryRecord(ctx context.Context, d models.Delivery) error {

	query := `
	INSERT INTO deliveries (id, user_id, notification_id, status, retry_count)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(ctx, query,
		d.ID,
		d.UserID,
		d.NotificationID,
		d.Status,
		d.RetryCount,
	)

	if err != nil {
		return err
	}

	return nil
}