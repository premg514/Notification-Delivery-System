package service

import "time"

func GetRetryDelay(retryCount int) time.Duration {
	switch retryCount {
	case 1:
		return 5 * time.Second
	case 2:
		return 20 * time.Second
	case 3:
		return 60 * time.Second
	default:
		return 0
	}
}
