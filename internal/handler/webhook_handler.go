package handler

import (
	"encoding/json"
	"net/http"

	"idempotent-webhook-processor/internal/domain"
)

type WebhookHandler struct {
	useCase domain.WebhookUseCase
}

func NewWebhookHandler(uc domain.WebhookUseCase) *WebhookHandler {
	return &WebhookHandler{useCase: uc}
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload domain.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid json request payload", http.StatusBadRequest)
		return
	}

	// Payload Validation Rules
	if payload.EventID == "" || payload.ClientID == "" {
		http.Error(w, "Missing mandatory fields: event_id and client_id", http.StatusUnprocessableEntity)
		return
	}

	// Forward to processing engine
	err := h.useCase.ProcessWebhook(r.Context(), &payload)
	if err != nil {
		switch err {
		case domain.ErrDuplicateRequest:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"status": "ignored", "message": "Duplicate event transaction detected"}`))
			return
		case domain.ErrQueueFull:
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		default:
			http.Error(w, "Internal server processing failure", http.StatusInternalServerError)
			return
		}
	}

	// Return fast 202 Accepted status to client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status": "accepted", "message": "Payload queued safely for background execution"}`))
}
