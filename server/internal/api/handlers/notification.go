package handlers

import (
	"errors"
	"net/http"
	"strings"

	"notification-system/internal/domain/models"
	"notification-system/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(service *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

type NotificationRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

func (h *NotificationHandler) SendNotification(c *gin.Context) {
	if c.Request.Method == http.MethodOptions {
		c.Status(http.StatusNoContent)
		return
	}

	var req NotificationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Message = strings.TrimSpace(req.Message)
	if req.Title == "" || req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "title and message are required",
		})
		return
	}

	notification := models.Notification{
		ID:      uuid.New().String(),
		Title:   req.Title,
		Message: req.Message,
	}

	err := h.service.CreateNotification(c.Request.Context(), notification)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to create notification",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "notification created",
	})
}

func (h *NotificationHandler) GetNotification(c *gin.Context) {

	id := c.Param("id")

	notification, err := h.service.GetNotification(c.Request.Context(), id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "notification not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to fetch notification",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, notification)
}
