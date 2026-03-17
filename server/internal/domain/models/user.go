package models

import "time"

type User struct {
	ID          string
	Email       string
	DeviceToken string
	Department  Department
	CreatedAt   time.Time
}
