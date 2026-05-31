package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
)

type RawEvent struct {
	ID        int64           `json:"id"`
	Source    string          `json:"source"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type Alert struct {
	ID              int64             `json:"id"`
	Fingerprint     string            `json:"fingerprint"`
	Status          AlertStatus       `json:"status"`
	Severity        string            `json:"severity"`
	Environment     string            `json:"environment"`
	Service         string            `json:"service"`
	Resource        string            `json:"resource"`
	Team            string            `json:"team"`
	Summary         string            `json:"summary"`
	Description     string            `json:"description"`
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
	StartsAt        time.Time         `json:"starts_at"`
	EndsAt          *time.Time        `json:"ends_at,omitempty"`
	LastSeen        time.Time         `json:"last_seen"`
	OccurrenceCount int               `json:"occurrence_count"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Source          string            `json:"source"`
}

// GenerateFingerprint creates a deterministic hash of the alert based on stable fields.
func (a *Alert) GenerateFingerprint() string {
	h := sha256.New()
	// Recommended fingerprint fields:
	// source, alert name, service, environment, resource, severity
	
	// We use the "alertname" from labels if available, otherwise we might need a Name field.
	// Looking at the Alert struct, it doesn't have a Name field, but Prometheus alerts 
	// usually have an "alertname" label.
	
	alertName := a.Labels["alertname"]
	
	fields := []string{
		a.Source,
		alertName,
		a.Service,
		a.Environment,
		a.Resource,
		a.Severity,
	}

	for _, f := range fields {
		h.Write([]byte(f))
		h.Write([]byte("|")) // Separator to avoid collisions
	}

	return hex.EncodeToString(h.Sum(nil))
}

type AlertEvent struct {
	ID        int64       `json:"id"`
	AlertID   int64       `json:"alert_id"`
	Status    AlertStatus `json:"status"`
	EventTime time.Time   `json:"event_time"`
	Details   string      `json:"details"`
	CreatedAt time.Time   `json:"created_at"`
}
