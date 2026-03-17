package service

import (
	"crypto/rand"
	"encoding/hex"
)

func NewID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "fallback-id"
	}
	return hex.EncodeToString(buffer)
}
