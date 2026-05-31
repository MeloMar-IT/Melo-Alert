package domain

import (
	"testing"
)

func TestAlert_GenerateFingerprint(t *testing.T) {
	tests := []struct {
		name     string
		alert    Alert
		expected string
	}{
		{
			name: "identical logical alerts",
			alert: Alert{
				Source:      "prometheus",
				Service:     "web",
				Environment: "prod",
				Resource:    "node-1",
				Severity:    "critical",
				Labels:      map[string]string{"alertname": "HighCPU"},
			},
		},
		{
			name: "different resource",
			alert: Alert{
				Source:      "prometheus",
				Service:     "web",
				Environment: "prod",
				Resource:    "node-2",
				Severity:    "critical",
				Labels:      map[string]string{"alertname": "HighCPU"},
			},
		},
		{
			name: "different severity",
			alert: Alert{
				Source:      "prometheus",
				Service:     "web",
				Environment: "prod",
				Resource:    "node-1",
				Severity:    "warning",
				Labels:      map[string]string{"alertname": "HighCPU"},
			},
		},
		{
			name: "identical fields, extra labels (should be same)",
			alert: Alert{
				Source:      "prometheus",
				Service:     "web",
				Environment: "prod",
				Resource:    "node-1",
				Severity:    "critical",
				Labels:      map[string]string{"alertname": "HighCPU", "extra": "info"},
			},
		},
	}

	fingerprints := make(map[string]string)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := tt.alert.GenerateFingerprint()
			if fp == "" {
				t.Error("fingerprint should not be empty")
			}
			fingerprints[tt.name] = fp
		})
	}

	// Verify stability and differences
	if fingerprints["identical logical alerts"] != fingerprints["identical fields, extra labels (should be same)"] {
		t.Error("fingerprints should be identical for same logical fields even with extra labels")
	}

	if fingerprints["identical logical alerts"] == fingerprints["different resource"] {
		t.Error("fingerprints should be different for different resources")
	}

	if fingerprints["identical logical alerts"] == fingerprints["different severity"] {
		t.Error("fingerprints should be different for different severity")
	}
}

func TestAlert_DeterministicFingerprint(t *testing.T) {
	alert := Alert{
		Source:      "prometheus",
		Service:     "web",
		Environment: "prod",
		Resource:    "node-1",
		Severity:    "critical",
		Labels:      map[string]string{"alertname": "HighCPU"},
	}

	fp1 := alert.GenerateFingerprint()
	fp2 := alert.GenerateFingerprint()

	if fp1 != fp2 {
		t.Errorf("fingerprint generation should be deterministic: %s != %s", fp1, fp2)
	}
}
