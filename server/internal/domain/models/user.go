package models

import "time"

type User struct {
	ID          string
	Email       string
	DeviceToken string
	CreatedAt   time.Time
}