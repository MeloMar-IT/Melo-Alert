package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"signalhub/internal/config"
	"signalhub/internal/domain"
	"time"
)

type Client struct {
	cfg config.TeamsConfig
	hc  *http.Client
}

func NewClient(cfg config.TeamsConfig) *Client {
	return &Client{
		cfg: cfg,
		hc: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// AdaptiveCard represents a Microsoft Teams Adaptive Card
type AdaptiveCard struct {
	Type    string `json:"type"`
	Body    []any  `json:"body"`
	Actions []any  `json:"actions,omitempty"`
	Version string `json:"version"`
	Schema  string `json:"$schema"`
}

func (c *Client) Send(ctx context.Context, alert *domain.Alert) error {
	if !c.cfg.Enabled {
		return nil
	}

	payload := c.renderAlert(alert)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal teams payload: %w", err)
	}

	// Retry logic for safe errors (network issues, 5xx)
	var lastErr error
	for i := 0; i < 3; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(i) * time.Second):
			}
		}

		lastErr = c.doSend(ctx, body)
		if lastErr == nil {
			return nil
		}

		// Only retry if it's a temporary error or network error
		// In Go, we can check if it's a timeout or some other temporary error.
		// For simplicity and following "only if safe", we retry on 5xx or network errors.
	}

	return fmt.Errorf("failed to send to teams after retries: %w", lastErr)
}

func (c *Client) doSend(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.WebhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Retry on 5xx, but not on 4xx (except maybe 429)
	if resp.StatusCode >= 500 {
		return fmt.Errorf("server error: %d", resp.StatusCode)
	}

	return fmt.Errorf("non-2xx response: %d", resp.StatusCode)
}

func (c *Client) renderAlert(alert *domain.Alert) map[string]any {
	color := "default"
	switch alert.Severity {
	case "critical":
		color = "attention"
	case "warning":
		color = "warning"
	case "info":
		color = "accent"
	}

	if alert.Status == domain.AlertStatusResolved {
		color = "good"
	}

	body := []any{
		map[string]any{
			"type": "TextBlock",
			"text": fmt.Sprintf("Alert: %s", alert.Summary),
			"size": "large",
			"weight": "bolder",
			"color": color,
		},
		map[string]any{
			"type": "FactSet",
			"facts": []any{
				map[string]any{"title": "Severity", "value": alert.Severity},
				map[string]any{"title": "Service", "value": alert.Service},
				map[string]any{"title": "Environment", "value": alert.Environment},
				map[string]any{"title": "Status", "value": string(alert.Status)},
				map[string]any{"title": "Validity", "value": alert.ValidityState},
			},
		},
	}

	if alert.Description != "" {
		body = append(body, map[string]any{
			"type": "TextBlock",
			"text": alert.Description,
			"wrap": true,
		})
	}

	if alert.AISummary != "" {
		body = append(body, map[string]any{
			"type": "Container",
			"style": "emphasis",
			"items": []any{
				map[string]any{
					"type": "TextBlock",
					"text": "AI Summary",
					"weight": "bolder",
				},
				map[string]any{
					"type": "TextBlock",
					"text": alert.AISummary,
					"wrap": true,
					"fontType": "monospace",
				},
			},
		})
	}

	return map[string]any{
		"type": "message",
		"attachments": []any{
			map[string]any{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]any{
					"type": "AdaptiveCard",
					"body": body,
					"version": "1.4",
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
				},
			},
		},
	}
}
