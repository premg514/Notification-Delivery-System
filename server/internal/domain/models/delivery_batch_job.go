package models

type DeliveryBatchJob struct {
	NotificationID string            `json:"notification_id"`
	Items          []DeliveryAttempt `json:"items"`
}

type DeliveryAttempt struct {
	DeliveryID     string `json:"delivery_id"`
	UserID         string `json:"user_id"`
	Title          string `json:"title"`
	Message        string `json:"message"`
	Priority       string `json:"priority"`
	RetryCount     int    `json:"retry_count"`
	IdempotencyKey string `json:"idempotency_key"`
}
