package models

import "time"

type Notification struct {
	ID                string
	Title             string
	Message           string
	TargetDepartment  Department
	Priority          string
	IdempotencyKey    string
	CreatedAt         time.Time
	TargetRecipients  int
	QueuedDeliveries  int
	SentDeliveries    int
	PendingDeliveries int
}
