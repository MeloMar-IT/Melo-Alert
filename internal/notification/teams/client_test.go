package teams

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"signalhub/internal/config"
	"signalhub/internal/domain"
	"testing"
	"time"
)

func TestClient_Send(t *testing.T) {
	alert := &domain.Alert{
		Summary:       "High CPU",
		Severity:      "critical",
		Service:       "auth-service",
		Environment:   "production",
		Status:        domain.AlertStatusFiring,
		ValidityState: "valid",
		AISummary:     "CPU is high due to many requests",
	}

	tests := []struct {
		name           string
		enabled        bool
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name:    "Success",
			enabled: true,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:    "Disabled",
			enabled: false,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				t.Error("Should not be called")
			},
			wantErr: false,
		},
		{
			name:    "Server Error Retry and Success",
			enabled: true,
			serverResponse: func() func(w http.ResponseWriter, r *http.Request) {
				count := 0
				return func(w http.ResponseWriter, r *http.Request) {
					count++
					if count == 1 {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusOK)
				}
			}(),
			wantErr: false,
		},
		{
			name:    "Client Error No Retry",
			enabled: true,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			cfg := config.TeamsConfig{
				Enabled:    tt.enabled,
				WebhookURL: server.URL,
				Timeout:    1 * time.Second,
			}

			client := NewClient(cfg)
			err := client.Send(context.Background(), alert)
			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_RenderAlert(t *testing.T) {
	alert := &domain.Alert{
		Summary:       "High CPU",
		Severity:      "critical",
		Service:       "auth-service",
		Environment:   "production",
		Status:        domain.AlertStatusFiring,
		ValidityState: "valid",
		AISummary:     "CPU is high",
	}

	cfg := config.TeamsConfig{Enabled: true}
	client := NewClient(cfg)
	payload := client.renderAlert(alert)

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	// Basic validation of payload structure
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if m["type"] != "message" {
		t.Errorf("Expected type message, got %v", m["type"])
	}

	attachments := m["attachments"].([]any)
	if len(attachments) != 1 {
		t.Errorf("Expected 1 attachment, got %v", len(attachments))
	}
}
