package domain

import (
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
}

type AlertEvent struct {
	ID        int64       `json:"id"`
	AlertID   int64       `json:"alert_id"`
	Status    AlertStatus `json:"status"`
	EventTime time.Time   `json:"event_time"`
	Details   string      `json:"details"`
	CreatedAt time.Time   `json:"created_at"`
}
