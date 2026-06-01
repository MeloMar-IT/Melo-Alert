package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"signalhub/internal/domain"
	"signalhub/internal/notification/servicenow"
	"signalhub/internal/notification/teams"
)

type WebhookHandler struct {
	repo            domain.Repository
	logger          *slog.Logger
	teamsClient     *teams.Client
	servicenowClient *servicenow.Client
}

func NewWebhookHandler(repo domain.Repository, logger *slog.Logger, teamsClient *teams.Client, servicenowClient *servicenow.Client) *WebhookHandler {
	return &WebhookHandler{
		repo:            repo,
		logger:          logger,
		teamsClient:     teamsClient,
		servicenowClient: servicenowClient,
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

		// Send to Teams if enabled
		if h.teamsClient != nil {
			go func(a domain.Alert) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := h.teamsClient.Send(ctx, &a); err != nil {
					h.logger.Error("failed to send alert to teams", "error", err, "fingerprint", a.Fingerprint)
				}
			}(*alert)
		}

		// Send to ServiceNow if enabled
		if h.servicenowClient != nil {
			go func(a domain.Alert) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				// Check if incident already exists
				existing, err := h.servicenowClient.GetByCorrelationID(ctx, a.Fingerprint)
				if err != nil {
					h.logger.Error("failed to check for existing servicenow incident", "error", err, "fingerprint", a.Fingerprint)
					return
				}

				if existing != nil {
					// Update existing incident
					note := fmt.Sprintf("Alert status: %s\nSummary: %s", a.Status, a.Summary)
					if err := h.servicenowClient.AddWorkNotes(ctx, existing.SysID, note); err != nil {
						h.logger.Error("failed to update servicenow incident", "error", err, "sys_id", existing.SysID)
					}
				} else {
					// Create new incident
					incident := h.servicenowClient.MapAlertToIncident(&a)
					created, err := h.servicenowClient.Create(ctx, incident)
					if err != nil {
						h.logger.Error("failed to create servicenow incident", "error", err, "fingerprint", a.Fingerprint)
					} else {
						h.logger.Info("created servicenow incident", "number", created.Number, "sys_id", created.SysID)
					}
				}
			}(*alert)
		}
	}

	w.WriteHeader(http.StatusAccepted)
}
