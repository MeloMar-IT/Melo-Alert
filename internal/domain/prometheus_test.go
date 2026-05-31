package domain

import (
	"testing"
	"time"
)

func TestGenerateFingerprint(t *testing.T) {
	labels1 := map[string]string{
		"alertname": "TestAlert",
		"service":   "web",
		"severity":  "critical",
	}
	labels2 := map[string]string{
		"severity":  "critical",
		"service":   "web",
		"alertname": "TestAlert",
	}
	labels3 := map[string]string{
		"alertname": "TestAlert",
		"service":   "api",
		"severity":  "critical",
	}

	fp1 := GenerateFingerprint(labels1)
	fp2 := GenerateFingerprint(labels2)
	fp3 := GenerateFingerprint(labels3)

	if fp1 == "" {
		t.Fatal("fingerprint should not be empty")
	}
	if fp1 != fp2 {
		t.Errorf("fingerprints for same labels (different order) should be identical: %s != %s", fp1, fp2)
	}
	if fp1 == fp3 {
		t.Errorf("fingerprints for different labels should be different: %s == %s", fp1, fp3)
	}
}

func TestNormalizePrometheusAlert(t *testing.T) {
	startsAt := time.Now().Add(-1 * time.Hour)
	pa := PrometheusAlert{
		Status: "firing",
		Labels: map[string]string{
			"alertname":   "HighCpu",
			"service":     "order-api",
			"severity":    "warning",
			"environment": "prod",
			"instance":    "node-1",
		},
		Annotations: map[string]string{
			"summary":     "High CPU usage on node-1",
			"description": "CPU usage is above 90%",
		},
		StartsAt: startsAt,
	}

	alert := NormalizePrometheusAlert(pa)

	if alert.Status != AlertStatusFiring {
		t.Errorf("expected status firing, got %s", alert.Status)
	}
	if alert.Service != "order-api" {
		t.Errorf("expected service order-api, got %s", alert.Service)
	}
	if alert.Severity != "warning" {
		t.Errorf("expected severity warning, got %s", alert.Severity)
	}
	if alert.Environment != "prod" {
		t.Errorf("expected environment prod, got %s", alert.Environment)
	}
	if alert.Resource != "node-1" {
		t.Errorf("expected resource node-1, got %s", alert.Resource)
	}
	if alert.Summary != "High CPU usage on node-1" {
		t.Errorf("expected summary, got %s", alert.Summary)
	}
	if alert.StartsAt != startsAt {
		t.Errorf("expected starts_at to be preserved")
	}
	if alert.OccurrenceCount != 1 {
		t.Errorf("expected occurrence_count 1, got %d", alert.OccurrenceCount)
	}
	if alert.Fingerprint == "" {
		t.Error("expected fingerprint to be generated")
	}
}

func TestMapPrometheusStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected AlertStatus
	}{
		{"firing", AlertStatusFiring},
		{"FIRING", AlertStatusFiring},
		{"resolved", AlertStatusResolved},
		{"RESOLVED", AlertStatusResolved},
		{"unknown", AlertStatusFiring}, // Default
	}

	for _, tt := range tests {
		got := MapPrometheusStatus(tt.input)
		if got != tt.expected {
			t.Errorf("MapPrometheusStatus(%s) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}
