package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"notification-system/internal/service"
)

type NotificationHandler struct {
	service *service.NotificationService
}

type sendNotificationRequest struct {
	Title         string   `json:"title"`
	Message       string   `json:"message"`
	TargetUserIDs []string `json:"target_users"`
	Priority      string   `json:"priority"`
	RequestID     string   `json:"request_id"`
}

func NewNotificationHandler(service *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req sendNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Message) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and message are required"})
		return
	}

	if len(req.TargetUserIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "target_users must contain at least one user"})
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(req.RequestID)
	}

	result, err := h.service.QueueNotification(r.Context(), service.CreateNotificationInput{
		Title:          req.Title,
		Message:        req.Message,
		TargetUserIDs:  req.TargetUserIDs,
		Priority:       req.Priority,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	statusCode := http.StatusAccepted
	if result.Duplicate {
		statusCode = http.StatusOK
	}

	writeJSON(w, statusCode, result)
}

func (h *NotificationHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
