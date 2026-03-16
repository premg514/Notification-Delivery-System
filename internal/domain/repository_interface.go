package domain

import "notification-system/internal/domain/models"

type NotificationRepository interface {
	CreateNotification(notification models.Notification) error
	GetNotificationByID(id string) (models.Notification, error)
}