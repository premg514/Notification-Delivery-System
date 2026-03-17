package api

import (
	"notification-system/internal/api/handlers"
	"notification-system/internal/api/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(notificationHandler *handlers.NotificationHandler, rateLimiter *middleware.RateLimiter) *gin.Engine {
	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.Metrics())
	router.GET("/health", notificationHandler.Health)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := router.Group("/")
	api.Use(rateLimiter.Middleware())
	api.POST("/send-notification", notificationHandler.SendNotification)
	api.GET("/notifications/recent", notificationHandler.ListRecentNotifications)

	return router
}
