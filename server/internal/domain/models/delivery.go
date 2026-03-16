package models

import "time"

type Delivery struct {
	ID             string
	UserID         string
	NotificationID string
	Status         string
	RetryCount     int
	DeliveredAt    time.Time
}