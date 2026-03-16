package service

import (
	"context"
	"notification-system/internal/domain"
	"notification-system/internal/domain/models"
)

type NotificationService struct {
	repo domain.NotificationRepository
}

func NewNotificationService(repo domain.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) CreateNotification(ctx context.Context, n models.Notification) error {
	return s.repo.CreateNotification(ctx, n)
}