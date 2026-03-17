package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"notification-system/internal/domain/models"
	"notification-system/internal/service"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service *service.NotificationService
}

type sendNotificationRequest struct {
	Title            string `json:"title"`
	Message          string `json:"message"`
	TargetDepartment string `json:"target_department"`
	Priority         string `json:"priority"`
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

	targetDepartment, ok := models.ParseDepartment(req.TargetDepartment)
	if !ok {
		c.JSON(http.StatusBadRequest, map[string]string{
			"error": "target_department must be one of CSE, ECE, ME, CIVIL, EEE",
		})
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "Idempotency-Key header is required"})
		return
	}

	result, err := h.service.QueueNotification(c.Request.Context(), service.CreateNotificationInput{
		Title:            req.Title,
		Message:          req.Message,
		TargetDepartment: targetDepartment,
		Priority:         req.Priority,
		IdempotencyKey:   idempotencyKey,
	})
	if err != nil {
		handleServiceError(c, err)
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

func (h *NotificationHandler) ListRecentNotifications(c *gin.Context) {
	limit := 10
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
			return
		}
		limit = parsedLimit
	}

	notifications, err := h.service.ListRecentNotifications(c.Request.Context(), limit)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		c.JSON(http.StatusGatewayTimeout, map[string]string{"error": "request timed out"})
	case errors.Is(err, context.Canceled):
		c.JSON(http.StatusRequestTimeout, map[string]string{"error": "request canceled"})
	case errors.Is(err, service.ErrNoUsersInDepartment):
		c.JSON(http.StatusBadRequest, map[string]string{"error": "no users found for target_department"})
	default:
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}
