package models

type FanoutJob struct {
	NotificationID string   `json:"notification_id"`
	Title          string   `json:"title"`
	Message        string   `json:"message"`
	Priority       string   `json:"priority"`
	TargetUserIDs  []string `json:"target_user_ids"`
	IdempotencyKey string   `json:"idempotency_key"`
}
