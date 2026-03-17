package handlers

import (
	"net/http"
	"strings"

	"notification-system/internal/service"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service *service.NotificationService
}

type sendNotificationRequest struct {
	Title         string   `json:"title"`
	Message       string   `json:"message"`
	TargetUserIDs []string `json:"target_users"`
	Priority      string   `json:"priority"`
}

func NewNotificationHandler(service *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req sendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "title and message are required"})
		return
	}

	if len(req.TargetUserIDs) == 0 {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "target_users must contain at least one user"})
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "Idempotency-Key header is required"})
		return
	}

	result, err := h.service.QueueNotification(c.Request.Context(), service.CreateNotificationInput{
		Title:          req.Title,
		Message:        req.Message,
		TargetUserIDs:  req.TargetUserIDs,
		Priority:       req.Priority,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	statusCode := http.StatusAccepted
	if result.Duplicate {
		statusCode = http.StatusOK
	}

	c.JSON(statusCode, result)
}

func (h *NotificationHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
