package models

import "time"

type Notification struct {
	ID             string
	Title          string
	Message        string
	Priority       string
	IdempotencyKey string
	CreatedAt      time.Time
}
