package api

import (
	"notification-system/internal/api/handlers"
	"notification-system/internal/api/middleware"
	"notification-system/internal/service"

	"github.com/gin-gonic/gin"
)

func SetupRouter(notificationService *service.NotificationService) *gin.Engine {

	router := gin.Default()
	router.Use(middleware.CORS())

	handler := handlers.NewNotificationHandler(notificationService)

	router.POST("/send-notification", handler.SendNotification)
	// router.POST("/send-notification/", handler.SendNotification)
	// router.POST("/api/send-notification", handler.SendNotification)
	// router.POST("/api/send-notification/", handler.SendNotification)
	// router.OPTIONS("/send-notification", handler.SendNotification)
	// router.OPTIONS("/send-notification/", handler.SendNotification)
	// router.OPTIONS("/api/send-notification", handler.SendNotification)
	// router.OPTIONS("/api/send-notification/", handler.SendNotification)
	router.GET("/notification/:id", handler.GetNotification)
	// router.GET("/api/notification/:id", handler.GetNotification)

	return router
}
