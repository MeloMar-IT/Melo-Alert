package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"signalhub/internal/domain"
)

func TestRepository(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5432/signalhub?sslmode=disable"
	}

	ctx := context.Background()
	repo, err := NewRepository(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	t.Run("StoreRawEvent", func(t *testing.T) {
		event := &domain.RawEvent{
			Source:    "prometheus",
			Payload:   json.RawMessage(`{"alert": "test"}`),
			CreatedAt: time.Now(),
		}

		err := repo.StoreRawEvent(ctx, event)
		if err != nil {
			t.Errorf("failed to store raw event: %v", err)
		}
		if event.ID == 0 {
			t.Error("expected event ID to be set")
		}
	})

	t.Run("UpsertAlert", func(t *testing.T) {
		alert := &domain.Alert{
			Fingerprint: "test-fingerprint",
			Status:      domain.AlertStatusFiring,
			Source:      "test",
			Labels:      map[string]string{"alertname": "TestAlert"},
			Annotations: map[string]string{"summary": "Test summary"},
			StartsAt:    time.Now().Add(-10 * time.Minute),
		}

		// Initial insert
		err := repo.UpsertAlert(ctx, alert)
		if err != nil {
			t.Fatalf("failed to insert alert: %v", err)
		}
		if alert.ID == 0 {
			t.Error("expected alert ID to be set")
		}
		initialID := alert.ID

		// Update by fingerprint
		alert.Status = domain.AlertStatusResolved
		endsAt := time.Now()
		alert.EndsAt = &endsAt

		err = repo.UpsertAlert(ctx, alert)
		if err != nil {
			t.Errorf("failed to update alert: %v", err)
		}
		if alert.ID != initialID {
			t.Errorf("expected ID to remain %d, got %d", initialID, alert.ID)
		}
		if alert.Status != domain.AlertStatusResolved {
			t.Errorf("expected status to be resolved, got %s", alert.Status)
		}
	})

	t.Run("AppendAlertEvent", func(t *testing.T) {
		// Create an alert first
		alert := &domain.Alert{
			Fingerprint: "event-test-fingerprint",
			Status:      domain.AlertStatusFiring,
			Source:      "test",
			Labels:      map[string]string{"alertname": "EventTestAlert"},
			StartsAt:    time.Now(),
		}
		err := repo.UpsertAlert(ctx, alert)
		if err != nil {
			t.Fatalf("failed to create alert for event test: %v", err)
		}

		event := &domain.AlertEvent{
			AlertID:   alert.ID,
			Status:    domain.AlertStatusFiring,
			EventTime: time.Now(),
			Details:   "Alert started firing",
			CreatedAt: time.Now(),
		}

		err = repo.AppendAlertEvent(ctx, event)
		if err != nil {
			t.Errorf("failed to append alert event: %v", err)
		}
		if event.ID == 0 {
			t.Error("expected event ID to be set")
		}
	})
	t.Run("UpsertAlertDeduplication", func(t *testing.T) {
		fp := "dedup-test-fingerprint"
		alert := &domain.Alert{
			Fingerprint: fp,
			Status:      domain.AlertStatusFiring,
			Labels:      map[string]string{"alertname": "DedupAlert"},
			StartsAt:    time.Now().Add(-10 * time.Minute),
			LastSeen:    time.Now().Add(-10 * time.Minute),
		}

		// First insert
		err := repo.UpsertAlert(ctx, alert)
		if err != nil {
			t.Fatalf("failed to insert alert: %v", err)
		}
		if alert.OccurrenceCount != 1 {
			t.Errorf("expected occurrence_count 1, got %d", alert.OccurrenceCount)
		}

		// Second insert with same fingerprint
		alert2 := &domain.Alert{
			Fingerprint: fp,
			Status:      domain.AlertStatusFiring,
			Labels:      map[string]string{"alertname": "DedupAlert"},
			StartsAt:    time.Now().Add(-10 * time.Minute),
			LastSeen:    time.Now(),
		}

		err = repo.UpsertAlert(ctx, alert2)
		if err != nil {
			t.Fatalf("failed to upsert alert: %v", err)
		}
		if alert2.OccurrenceCount != 2 {
			t.Errorf("expected occurrence_count 2, got %d", alert2.OccurrenceCount)
		}
		if alert2.ID != alert.ID {
			t.Errorf("expected same ID, got %d != %d", alert2.ID, alert.ID)
		}
	})
}
