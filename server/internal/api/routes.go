package api

import (
	"net/http"

	"notification-system/internal/api/handlers"
	"notification-system/internal/api/middleware"
)

func NewRouter(notificationHandler *handlers.NotificationHandler, rateLimiter *middleware.RateLimiter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", notificationHandler.Health)
	mux.HandleFunc("POST /send-notification", notificationHandler.SendNotification)

	return rateLimiter.Middleware(mux)
}
