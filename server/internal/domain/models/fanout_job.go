package models

type FanoutJob struct {
	NotificationID   string     `json:"notification_id"`
	Title            string     `json:"title"`
	Message          string     `json:"message"`
	Priority         string     `json:"priority"`
	TargetDepartment Department `json:"target_department"`
	IdempotencyKey   string     `json:"idempotency_key"`
}
