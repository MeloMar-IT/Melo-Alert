package servicenow

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

func TestMapAlertToIncident(t *testing.T) {
	client := &Client{
		cfg: config.ServiceNowConfig{
			AssignmentGroup: "test-group",
		},
	}

	alert := &domain.Alert{
		Summary:     "Test Summary",
		Description: "Test Description",
		Severity:    "critical",
		Status:      domain.AlertStatusFiring,
		Fingerprint: "test-fingerprint",
		AISummary:   "AI summary here",
	}

	incident := client.MapAlertToIncident(alert)

	if incident.ShortDescription != alert.Summary {
		t.Errorf("expected summary %s, got %s", alert.Summary, incident.ShortDescription)
	}
	if incident.Severity != "1" {
		t.Errorf("expected severity 1, got %s", incident.Severity)
	}
	if incident.CorrelationID != alert.Fingerprint {
		t.Errorf("expected correlation_id %s, got %s", alert.Fingerprint, incident.CorrelationID)
	}
	if incident.AssignmentGroup != "test-group" {
		t.Errorf("expected assignment group test-group, got %s", incident.AssignmentGroup)
	}
}

func TestServiceNowClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == "POST" && r.URL.Path == "/api/now/table/incident":
			var inc Incident
			json.NewDecoder(r.Body).Decode(&inc)
			inc.SysID = "sys123"
			inc.Number = "INC001"
			json.NewEncoder(w).Encode(map[string]any{"result": inc})

		case r.Method == "GET" && r.URL.Path == "/api/now/table/incident":
			q := r.URL.Query().Get("sysparm_query")
			if q == "correlation_id=test-fp" {
				inc := Incident{SysID: "sys123", Number: "INC001", CorrelationID: "test-fp"}
				json.NewEncoder(w).Encode(map[string]any{"result": []Incident{inc}})
			} else {
				json.NewEncoder(w).Encode(map[string]any{"result": []Incident{}})
			}

		case r.Method == "PUT" && r.URL.Path == "/api/now/table/incident/sys123":
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := config.ServiceNowConfig{
		Enabled:     true,
		InstanceURL: server.URL,
		User:        "user",
		Password:    "pass",
		Timeout:     5 * time.Second,
	}
	client := NewClient(cfg)
	ctx := context.Background()

	// Test GetByCorrelationID
	inc, err := client.GetByCorrelationID(ctx, "test-fp")
	if err != nil {
		t.Fatalf("GetByCorrelationID failed: %v", err)
	}
	if inc == nil || inc.Number != "INC001" {
		t.Fatalf("expected incident INC001, got %v", inc)
	}

	// Test Create
	newInc := &Incident{ShortDescription: "New Inc"}
	created, err := client.Create(ctx, newInc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.SysID != "sys123" {
		t.Fatalf("expected sys_id sys123, got %s", created.SysID)
	}

	// Test Update
	err = client.AddWorkNotes(ctx, "sys123", "some notes")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
}
