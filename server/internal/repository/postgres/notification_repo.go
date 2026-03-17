package postgres

import (
	"context"
	"errors"
	"notification-system/internal/domain/models"
	"time"

	"github.com/jackc/pgx/v5"
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
	_, err := r.db.Exec(ctx, `
		INSERT INTO notifications (id, title, message, target_department, priority, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, n.ID, n.Title, n.Message, string(n.TargetDepartment), n.Priority, nullableString(n.IdempotencyKey), n.CreatedAt.UTC())
	return err
}

func (r *NotificationRepository) CreateNotificationIfAbsent(ctx context.Context, n models.Notification) (models.Notification, bool, error) {
	var notification models.Notification
	err := r.db.QueryRow(ctx, `
		INSERT INTO notifications (id, title, message, target_department, priority, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (idempotency_key)
		DO UPDATE SET idempotency_key = notifications.idempotency_key
		RETURNING id, title, message, target_department, priority, COALESCE(idempotency_key, ''), created_at
	`, n.ID, n.Title, n.Message, string(n.TargetDepartment), n.Priority, nullableString(n.IdempotencyKey), n.CreatedAt.UTC()).Scan(
		&notification.ID,
		&notification.Title,
		&notification.Message,
		&notification.TargetDepartment,
		&notification.Priority,
		&notification.IdempotencyKey,
		&notification.CreatedAt,
	)
	if err != nil {
		return models.Notification{}, false, err
	}

	return notification, notification.ID == n.ID, nil
}

func (r *NotificationRepository) CreateNotificationWithDeliveries(ctx context.Context, n models.Notification, deliveries []models.Delivery) (err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err = tx.Exec(ctx, `
		INSERT INTO notifications (id, title, message, target_department, priority, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, n.ID, n.Title, n.Message, string(n.TargetDepartment), n.Priority, nullableString(n.IdempotencyKey), n.CreatedAt.UTC()); err != nil {
		return err
	}

	if len(deliveries) > 0 {
		rows := make([][]any, 0, len(deliveries))
		for _, delivery := range deliveries {
			rows = append(rows, []any{
				delivery.ID,
				delivery.UserID,
				delivery.NotificationID,
				delivery.Status,
				delivery.RetryCount,
				delivery.DeliveredAt,
				nullableString(delivery.LastError),
				delivery.UpdatedAt.UTC(),
			})
		}

		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"deliveries"},
			[]string{"id", "user_id", "notification_id", "status", "retry_count", "delivered_at", "last_error", "updated_at"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Fetch notification by ID
func (r *NotificationRepository) GetNotificationByID(ctx context.Context, id string) (models.Notification, error) {
	var notification models.Notification
	err := r.db.QueryRow(ctx, `
		SELECT id, title, message, target_department, priority, COALESCE(idempotency_key, ''), created_at
		FROM notifications
		WHERE id = $1
	`, id).Scan(
		&notification.ID,
		&notification.Title,
		&notification.Message,
		&notification.TargetDepartment,
		&notification.Priority,
		&notification.IdempotencyKey,
		&notification.CreatedAt,
	)
	return notification, err
}

func (r *NotificationRepository) GetNotificationByIdempotencyKey(ctx context.Context, key string) (models.Notification, error) {
	var notification models.Notification
	err := r.db.QueryRow(ctx, `
		SELECT id, title, message, target_department, priority, COALESCE(idempotency_key, ''), created_at
		FROM notifications
		WHERE idempotency_key = $1
	`, key).Scan(
		&notification.ID,
		&notification.Title,
		&notification.Message,
		&notification.TargetDepartment,
		&notification.Priority,
		&notification.IdempotencyKey,
		&notification.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Notification{}, err
	}
	return notification, err
}

func (r *NotificationRepository) ListRecentNotifications(ctx context.Context, limit int) ([]models.Notification, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			n.id,
			n.title,
			n.message,
			n.target_department,
			n.priority,
			COALESCE(n.idempotency_key, ''),
			n.created_at,
			COUNT(d.id)::INT AS queued_deliveries
		FROM notifications n
		LEFT JOIN deliveries d ON d.notification_id = n.id
		GROUP BY n.id, n.title, n.message, n.target_department, n.priority, n.idempotency_key, n.created_at
		ORDER BY n.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := make([]models.Notification, 0, limit)
	for rows.Next() {
		var notification models.Notification
		if err := rows.Scan(
			&notification.ID,
			&notification.Title,
			&notification.Message,
			&notification.TargetDepartment,
			&notification.Priority,
			&notification.IdempotencyKey,
			&notification.CreatedAt,
			&notification.QueuedDeliveries,
		); err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *NotificationRepository) ListUserIDsByDepartment(ctx context.Context, department models.Department) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id
		FROM users
		WHERE department = $1
		ORDER BY created_at, id
	`, string(department))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userIDs := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userIDs, nil
}

// Update delivery status
func (r *NotificationRepository) UpdateDeliveryStatus(ctx context.Context, deliveryID string, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE deliveries
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, deliveryID, status)
	return err
}

func (r *NotificationRepository) UpdateDeliveryStatusWithRetry(ctx context.Context, deliveryID string, status string, retryCount int, deliveredAt *time.Time, lastError string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE deliveries
		SET status = $2,
			retry_count = $3,
			delivered_at = $4,
			last_error = $5,
			updated_at = NOW()
		WHERE id = $1
	`, deliveryID, status, retryCount, deliveredAt, nullableString(lastError))
	return err
}

// Create delivery record
func (r *NotificationRepository) CreateDeliveryRecord(ctx context.Context, d models.Delivery) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO deliveries (id, user_id, notification_id, status, retry_count, delivered_at, last_error, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, d.ID, d.UserID, d.NotificationID, d.Status, d.RetryCount, d.DeliveredAt, nullableString(d.LastError), d.UpdatedAt.UTC())
	return err
}

func (r *NotificationRepository) CreateDeliveriesIfAbsent(ctx context.Context, deliveries []models.Delivery) ([]models.Delivery, error) {
	if len(deliveries) == 0 {
		return nil, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err = tx.Exec(ctx, `
		CREATE TEMP TABLE delivery_stage (
			id UUID,
			user_id UUID,
			notification_id UUID,
			status TEXT,
			retry_count INT,
			delivered_at TIMESTAMP,
			last_error TEXT,
			updated_at TIMESTAMP
		) ON COMMIT DROP
	`); err != nil {
		return nil, err
	}

	rows := make([][]any, 0, len(deliveries))
	for _, delivery := range deliveries {
		rows = append(rows, []any{
			delivery.ID,
			delivery.UserID,
			delivery.NotificationID,
			delivery.Status,
			delivery.RetryCount,
			delivery.DeliveredAt,
			nullableString(delivery.LastError),
			delivery.UpdatedAt.UTC(),
		})
	}

	if _, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"delivery_stage"},
		[]string{"id", "user_id", "notification_id", "status", "retry_count", "delivered_at", "last_error", "updated_at"},
		pgx.CopyFromRows(rows),
	); err != nil {
		return nil, err
	}

	insertedRows, err := tx.Query(ctx, `
		INSERT INTO deliveries (id, user_id, notification_id, status, retry_count, delivered_at, last_error, updated_at)
		SELECT id, user_id, notification_id, status, retry_count, delivered_at, last_error, updated_at
		FROM delivery_stage
		ON CONFLICT (notification_id, user_id) DO NOTHING
		RETURNING id, user_id, notification_id, status, retry_count, delivered_at, COALESCE(last_error, ''), updated_at
	`)
	if err != nil {
		return nil, err
	}
	defer insertedRows.Close()

	inserted := make([]models.Delivery, 0, len(deliveries))
	for insertedRows.Next() {
		var delivery models.Delivery
		if scanErr := insertedRows.Scan(
			&delivery.ID,
			&delivery.UserID,
			&delivery.NotificationID,
			&delivery.Status,
			&delivery.RetryCount,
			&delivery.DeliveredAt,
			&delivery.LastError,
			&delivery.UpdatedAt,
		); scanErr != nil {
			err = scanErr
			return nil, err
		}
		inserted = append(inserted, delivery)
	}
	if err = insertedRows.Err(); err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return inserted, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
