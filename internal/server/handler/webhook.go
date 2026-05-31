package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"signalhub/internal/domain"
)

type WebhookHandler struct {
	repo   domain.Repository
	logger *slog.Logger
}

func NewWebhookHandler(repo domain.Repository, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		repo:   repo,
		logger: logger,
	}
}

func (h *WebhookHandler) HandleGeneric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", "error", err)
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}

	// Validate JSON
	if !json.Valid(body) {
		h.logger.Warn("received malformed JSON")
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	event := &domain.RawEvent{
		Source:    "generic",
		Payload:   json.RawMessage(body),
		CreatedAt: time.Now(),
	}

	if err := h.repo.StoreRawEvent(r.Context(), event); err != nil {
		h.logger.Error("failed to store raw event", "error", err)
		http.Error(w, "failed to store event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *WebhookHandler) HandlePrometheus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", "error", err)
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}

	var payload domain.PrometheusWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Warn("failed to unmarshal prometheus payload", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Store raw event
	event := &domain.RawEvent{
		Source:    "prometheus",
		Payload:   json.RawMessage(body),
		CreatedAt: time.Now(),
	}
	if err := h.repo.StoreRawEvent(r.Context(), event); err != nil {
		h.logger.Error("failed to store raw event", "error", err)
		// Continue even if raw storage fails? Usually better to be safe.
	}

	for _, pa := range payload.Alerts {
		alert := domain.NormalizePrometheusAlert(pa)

		if err := h.repo.UpsertAlert(r.Context(), alert); err != nil {
			h.logger.Error("failed to upsert alert", "error", err, "fingerprint", alert.Fingerprint)
			continue
		}

		alertEvent := &domain.AlertEvent{
			AlertID:   alert.ID,
			Status:    alert.Status,
			EventTime: pa.StartsAt,
			CreatedAt: time.Now(),
		}
		if err := h.repo.AppendAlertEvent(r.Context(), alertEvent); err != nil {
			h.logger.Error("failed to append alert event", "error", err, "alert_id", alert.ID)
		}
	}

	w.WriteHeader(http.StatusAccepted)
}
