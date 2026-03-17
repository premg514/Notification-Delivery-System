package api

import (
	"notification-system/internal/api/handlers"
	"notification-system/internal/api/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter(notificationHandler *handlers.NotificationHandler, rateLimiter *middleware.RateLimiter) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(rateLimiter.Middleware())
	router.GET("/health", notificationHandler.Health)
	router.POST("/send-notification", notificationHandler.SendNotification)

	return router
}
