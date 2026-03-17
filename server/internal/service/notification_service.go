package service

import (
	"context"
	"errors"
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

var ErrNoUsersInDepartment = errors.New("no users found for target department")

type CreateNotificationInput struct {
	Title            string
	Message          string
	TargetDepartment models.Department
	Priority         string
	IdempotencyKey   string
}

type CreateNotificationResult struct {
	NotificationID   string    `json:"notification_id"`
	Status           string    `json:"status"`
	QueuedDeliveries int       `json:"queued_deliveries"`
	TargetDepartment string    `json:"target_department"`
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
	userIDs, err := s.repo.ListUserIDsByDepartment(ctx, input.TargetDepartment)
	if err != nil {
		return CreateNotificationResult{}, err
	}
	if len(userIDs) == 0 {
		return CreateNotificationResult{}, ErrNoUsersInDepartment
	}

	notification, created, err := s.repo.CreateNotificationIfAbsent(ctx, models.Notification{
		ID:               NewID(),
		Title:            strings.TrimSpace(input.Title),
		Message:          strings.TrimSpace(input.Message),
		TargetDepartment: input.TargetDepartment,
		Priority:         normalizePriority(input.Priority),
		IdempotencyKey:   idempotencyKey,
		CreatedAt:        now,
	})
	if err != nil {
		return CreateNotificationResult{}, err
	}

	if !created {
		return CreateNotificationResult{
			NotificationID:   notification.ID,
			Status:           "already_sent",
			QueuedDeliveries: len(userIDs),
			TargetDepartment: string(input.TargetDepartment),
			Duplicate:        true,
			CreatedAt:        notification.CreatedAt,
		}, nil
	}

	if err := s.publisher.PublishFanout(ctx, models.FanoutJob{
		NotificationID:   notification.ID,
		Title:            notification.Title,
		Message:          notification.Message,
		Priority:         notification.Priority,
		TargetDepartment: input.TargetDepartment,
		IdempotencyKey:   idempotencyKey,
	}); err != nil {
		return CreateNotificationResult{}, err
	}

	return CreateNotificationResult{
		NotificationID:   notification.ID,
		Status:           "queue",
		QueuedDeliveries: len(userIDs),
		TargetDepartment: string(input.TargetDepartment),
		CreatedAt:        notification.CreatedAt,
	}, nil
}

type RecentNotification struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	TargetDepartment string    `json:"target_department"`
	Priority         string    `json:"priority"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	QueuedDeliveries int       `json:"queued_deliveries"`
}

func (s *NotificationService) ListRecentNotifications(ctx context.Context, limit int) ([]RecentNotification, error) {
	notifications, err := s.repo.ListRecentNotifications(ctx, limit)
	if err != nil {
		return nil, err
	}

	result := make([]RecentNotification, 0, len(notifications))
	for _, notification := range notifications {
		result = append(result, RecentNotification{
			ID:               notification.ID,
			Title:            notification.Title,
			TargetDepartment: string(notification.TargetDepartment),
			Priority:         notification.Priority,
			Status:           deriveNotificationStatus(notification),
			CreatedAt:        notification.CreatedAt,
			QueuedDeliveries: notification.QueuedDeliveries,
		})
	}

	return result, nil
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

func deriveNotificationStatus(notification models.Notification) string {
	if notification.QueuedDeliveries == 0 {
		return "process"
	}

	if notification.PendingDeliveries > 0 {
		return "queue"
	}

	if notification.SentDeliveries >= notification.QueuedDeliveries {
		return "sent"
	}

	return "queue"
}
