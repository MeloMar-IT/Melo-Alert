package servicenow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"signalhub/internal/config"
	"signalhub/internal/domain"
)

// ServiceNow incident states
const (
	StateNew        = "1"
	StateInProgress = "2"
	StateOnHold     = "3"
	StateResolved   = "6"
	StateClosed     = "7"
	StateCanceled   = "8"
)

// Incident represents a ServiceNow Incident
type Incident struct {
	SysID             string `json:"sys_id,omitempty"`
	Number            string `json:"number,omitempty"`
	ShortDescription  string `json:"short_description"`
	Description       string `json:"description"`
	AssignmentGroup   string `json:"assignment_group,omitempty"`
	Severity          string `json:"severity,omitempty"`
	Urgency           string `json:"urgency,omitempty"`
	State             string `json:"state,omitempty"`
	WorkNotes         string `json:"work_notes,omitempty"`
	CorrelationID     string `json:"correlation_id,omitempty"`
}

type Client struct {
	cfg config.ServiceNowConfig
	hc  *http.Client
}

func NewClient(cfg config.ServiceNowConfig) *Client {
	return &Client{
		cfg: cfg,
		hc: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// MapAlertToIncident maps a domain Alert to a ServiceNow Incident
func (c *Client) MapAlertToIncident(alert *domain.Alert) *Incident {
	severity := "3" // 3 - Low
	switch alert.Severity {
	case "critical":
		severity = "1" // 1 - High
	case "warning":
		severity = "2" // 2 - Medium
	}

	state := StateNew
	if alert.Status == domain.AlertStatusResolved {
		state = StateResolved
	}

	description := alert.Description
	if alert.AISummary != "" {
		description = fmt.Sprintf("%s\n\nAI Summary:\n%s", description, alert.AISummary)
	}

	return &Incident{
		ShortDescription: alert.Summary,
		Description:      description,
		Severity:         severity,
		Urgency:          severity,
		State:            state,
		CorrelationID:    alert.Fingerprint,
		AssignmentGroup:  c.cfg.AssignmentGroup,
	}
}

// Create creates a new incident in ServiceNow
func (c *Client) Create(ctx context.Context, incident *Incident) (*Incident, error) {
	url := fmt.Sprintf("%s/api/now/table/incident", c.cfg.InstanceURL)
	
	body, err := json.Marshal(incident)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal incident: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result struct {
		Result Incident `json:"result"`
	}

	err = c.doRequest(req, &result)
	if err != nil {
		return nil, err
	}

	return &result.Result, nil
}

// Update updates an existing incident in ServiceNow
func (c *Client) Update(ctx context.Context, sysID string, updates map[string]any) error {
	url := fmt.Sprintf("%s/api/now/table/incident/%s", c.cfg.InstanceURL, sysID)

	body, err := json.Marshal(updates)
	if err != nil {
		return fmt.Errorf("failed to marshal updates: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	return c.doRequest(req, nil)
}

// AddWorkNotes adds work notes to an existing incident
func (c *Client) AddWorkNotes(ctx context.Context, sysID string, notes string) error {
	return c.Update(ctx, sysID, map[string]any{
		"work_notes": notes,
	})
}

// GetByCorrelationID searches for an incident by correlation ID
func (c *Client) GetByCorrelationID(ctx context.Context, correlationID string) (*Incident, error) {
	url := fmt.Sprintf("%s/api/now/table/incident?sysparm_query=correlation_id=%s&sysparm_limit=1", c.cfg.InstanceURL, correlationID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result struct {
		Result []Incident `json:"result"`
	}

	err = c.doRequest(req, &result)
	if err != nil {
		return nil, err
	}

	if len(result.Result) == 0 {
		return nil, nil
	}

	return &result.Result[0], nil
}

func (c *Client) doRequest(req *http.Request, result any) error {
	req.SetBasicAuth(c.cfg.User, c.cfg.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("servicenow api error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
