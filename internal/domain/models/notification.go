package models

import "time"

type Notification struct {
	ID        string
	Title     string
	Message   string
	CreatedAt time.Time
}