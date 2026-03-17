package service

import (
	"context"
	"notification-system/internal/domain"
	"notification-system/internal/domain/models"
	"strings"
	"time"
)

type NotificationService struct {
	repo      domain.NotificationRepository
	publisher QueuePublisher
}

type QueuePublisher interface {
	PublishFanout(ctx context.Context, job models.FanoutJob) error
}

type CreateNotificationInput struct {
	Title          string
	Message        string
	TargetUserIDs  []string
	Priority       string
	IdempotencyKey string
}

type CreateNotificationResult struct {
	NotificationID   string    `json:"notification_id"`
	Status           string    `json:"status"`
	QueuedDeliveries int       `json:"queued_deliveries"`
	Duplicate        bool      `json:"duplicate"`
	CreatedAt        time.Time `json:"created_at"`
}

func NewNotificationService(repo domain.NotificationRepository, publisher QueuePublisher) *NotificationService {
	return &NotificationService{repo: repo, publisher: publisher}
}

func (s *NotificationService) CreateNotification(ctx context.Context, n models.Notification) error {
	return s.repo.CreateNotification(ctx, n)
}

func (s *NotificationService) GetNotification(ctx context.Context, id string) (models.Notification, error) {
	return s.repo.GetNotificationByID(ctx, id)
}

func (s *NotificationService) QueueNotification(ctx context.Context, input CreateNotificationInput) (CreateNotificationResult, error) {
	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = NewID()
	}

	now := time.Now().UTC()
	userIDs := dedupeUserIDs(input.TargetUserIDs)
	notification, created, err := s.repo.CreateNotificationIfAbsent(ctx, models.Notification{
		ID:             NewID(),
		Title:          strings.TrimSpace(input.Title),
		Message:        strings.TrimSpace(input.Message),
		Priority:       normalizePriority(input.Priority),
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
	})
	if err != nil {
		return CreateNotificationResult{}, err
	}

	if !created {
		return CreateNotificationResult{
			NotificationID:   notification.ID,
			Status:           "already_queued",
			QueuedDeliveries: len(userIDs),
			Duplicate:        true,
			CreatedAt:        notification.CreatedAt,
		}, nil
	}

	if err := s.publisher.PublishFanout(ctx, models.FanoutJob{
		NotificationID: notification.ID,
		Title:          notification.Title,
		Message:        notification.Message,
		Priority:       notification.Priority,
		TargetUserIDs:  userIDs,
		IdempotencyKey: idempotencyKey,
	}); err != nil {
		return CreateNotificationResult{}, err
	}

	return CreateNotificationResult{
		NotificationID:   notification.ID,
		Status:           "queued",
		QueuedDeliveries: len(userIDs),
		CreatedAt:        notification.CreatedAt,
	}, nil
}

func dedupeUserIDs(userIDs []string) []string {
	seen := make(map[string]struct{}, len(userIDs))
	result := make([]string, 0, len(userIDs))

	for _, userID := range userIDs {
		trimmed := strings.TrimSpace(userID)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}

func normalizePriority(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "normal"
	}
}
