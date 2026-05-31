package domain

import (
	"strings"
	"time"
)

// PrometheusWebhookPayload represents the Alertmanager webhook payload.
type PrometheusWebhookPayload struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []PrometheusAlert `json:"alerts"`
}

// PrometheusAlert represents a single alert in the Alertmanager payload.
type PrometheusAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// NormalizePrometheusAlert maps a Prometheus alert to our internal Alert domain model.
func NormalizePrometheusAlert(pa PrometheusAlert) *Alert {
	alert := &Alert{
		Status:          MapPrometheusStatus(pa.Status),
		Labels:          pa.Labels,
		Annotations:     pa.Annotations,
		StartsAt:        pa.StartsAt,
		LastSeen:        time.Now(),
		OccurrenceCount: 1,
		Summary:         pa.Annotations["summary"],
		Description:     pa.Annotations["description"],
	}

	if !pa.EndsAt.IsZero() && pa.Status == "resolved" {
		alert.EndsAt = &pa.EndsAt
	}

	// Map labels to internal fields
	alert.Source = "prometheus"
	alert.Service = pa.Labels["service"]
	alert.Severity = pa.Labels["severity"]
	alert.Environment = pa.Labels["environment"]
	alert.Resource = pa.Labels["resource"]
	if alert.Resource == "" {
		alert.Resource = pa.Labels["instance"]
	}
	alert.Team = pa.Labels["team"]

	// Generate fingerprint if not provided or to ensure consistency
	alert.Fingerprint = alert.GenerateFingerprint()

	return alert
}

// MapPrometheusStatus converts Alertmanager status to internal AlertStatus.
func MapPrometheusStatus(status string) AlertStatus {
	switch strings.ToLower(status) {
	case "firing":
		return AlertStatusFiring
	case "resolved":
		return AlertStatusResolved
	default:
		return AlertStatusFiring
	}
}
