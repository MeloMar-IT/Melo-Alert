package server

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"signalhub/internal/config"
	"signalhub/internal/domain"
)

type mockRepository struct {
	storeRawEventFunc func(ctx context.Context, event *domain.RawEvent) error
}

func (m *mockRepository) StoreRawEvent(ctx context.Context, event *domain.RawEvent) error {
	return m.storeRawEventFunc(ctx, event)
}

func (m *mockRepository) UpsertAlert(ctx context.Context, alert *domain.Alert) error {
	return nil
}

func (m *mockRepository) AppendAlertEvent(ctx context.Context, event *domain.AlertEvent) error {
	return nil
}

func TestServer(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			WebhookToken: "test-token",
		},
		Server: config.ServerConfig{
			Address: ":8080",
		},
	}
	logger := slog.New(slog.NewJSONHandler(bytes.NewBuffer(nil), nil))
	
	t.Run("valid webhook request", func(t *testing.T) {
		repo := &mockRepository{
			storeRawEventFunc: func(ctx context.Context, event *domain.RawEvent) error {
				if string(event.Payload) != `{"foo":"bar"}` {
					t.Errorf("unexpected payload: %s", event.Payload)
				}
				return nil
			},
		}
		srv := New(cfg, logger, repo)

		req := httptest.NewRequest(http.MethodPost, "/webhooks/generic", bytes.NewBufferString(`{"foo":"bar"}`))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Errorf("expected status 202, got %d", rec.Code)
		}

		if rec.Header().Get("X-Request-ID") == "" {
			t.Error("expected X-Request-ID header to be present")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		repo := &mockRepository{}
		srv := New(cfg, logger, repo)

		req := httptest.NewRequest(http.MethodPost, "/webhooks/generic", bytes.NewBufferString(`{"foo":"bar"}`))
		req.Header.Set("Authorization", "Bearer wrong-token")
		
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		repo := &mockRepository{}
		srv := New(cfg, logger, repo)

		req := httptest.NewRequest(http.MethodPost, "/webhooks/generic", bytes.NewBufferString(`{"foo":`))
		req.Header.Set("Authorization", "Bearer test-token")
		
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("payload too large", func(t *testing.T) {
		repo := &mockRepository{}
		srv := New(cfg, logger, repo)

		// Create a payload larger than 1MB
		largePayload := make([]byte, 1024*1024+1)
		req := httptest.NewRequest(http.MethodPost, "/webhooks/generic", bytes.NewReader(largePayload))
		req.Header.Set("Authorization", "Bearer test-token")
		
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected status 413, got %d", rec.Code)
		}
	})

	t.Run("valid prometheus webhook request", func(t *testing.T) {
		repo := &mockRepository{
			storeRawEventFunc: func(ctx context.Context, event *domain.RawEvent) error {
				return nil
			},
		}
		srv := New(cfg, logger, repo)

		payload := `{
			"version": "4",
			"status": "firing",
			"alerts": [
				{
					"status": "firing",
					"labels": {
						"alertname": "TestAlert",
						"service": "test-service",
						"severity": "critical"
					},
					"annotations": {
						"summary": "Test summary"
					},
					"startsAt": "2024-05-31T10:00:00Z"
				}
			]
		}`

		req := httptest.NewRequest(http.MethodPost, "/webhooks/prometheus", bytes.NewBufferString(payload))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Errorf("expected status 202, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}
