package domain

import (
	"reflect"
	"testing"
)

func TestMatcher_Match(t *testing.T) {
	config := RoutingConfig{
		DefaultReceiver: "default",
		Routes: []Route{
			{
				Receiver: "critical-alerts",
				Match: Match{
					"severity": "critical",
				},
				Routes: []Route{
					{
						Receiver: "critical-prod",
						Match: Match{
							"environment": "production",
						},
					},
				},
			},
			{
				Receiver: "team-alpha",
				Match: Match{
					"team": "alpha",
				},
				Continue: true,
			},
			{
				Receiver: "service-x",
				Match: Match{
					"service": "x",
				},
			},
			{
				Receiver: "regex-route",
				MatchRE: Match{
					"service": "api-.*",
				},
			},
		},
	}

	matcher := NewMatcher(config)

	tests := []struct {
		name     string
		alert    *Alert
		expected []string
	}{
		{
			name: "Match critical production",
			alert: &Alert{
				Severity:    "critical",
				Environment: "production",
			},
			expected: []string{"critical-prod"},
		},
		{
			name: "Match critical non-production",
			alert: &Alert{
				Severity:    "critical",
				Environment: "staging",
			},
			expected: []string{"critical-alerts"},
		},
		{
			name: "Match team alpha with continue",
			alert: &Alert{
				Team:    "alpha",
				Service: "x",
			},
			expected: []string{"team-alpha", "service-x"},
		},
		{
			name: "Match service x",
			alert: &Alert{
				Service: "x",
			},
			expected: []string{"service-x"},
		},
		{
			name: "Match regex service",
			alert: &Alert{
				Service: "api-v1",
			},
			expected: []string{"regex-route"},
		},
		{
			name: "Match default",
			alert: &Alert{
				Severity: "warning",
				Service:  "unknown",
			},
			expected: []string{"default"},
		},
		{
			name: "Match labels",
			alert: &Alert{
				Labels: map[string]string{
					"app": "frontend",
				},
			},
			expected: []string{"default"}, // No route for app:frontend, so default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matcher.Match(tt.alert); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Matcher.Match() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMatcher_MatchLabels(t *testing.T) {
	config := RoutingConfig{
		DefaultReceiver: "default",
		Routes: []Route{
			{
				Receiver: "frontend-team",
				Match: Match{
					"app": "frontend",
				},
			},
		},
	}

	matcher := NewMatcher(config)

	alert := &Alert{
		Labels: map[string]string{
			"app": "frontend",
		},
	}

	expected := []string{"frontend-team"}
	if got := matcher.Match(alert); !reflect.DeepEqual(got, expected) {
		t.Errorf("Matcher.Match() = %v, want %v", got, expected)
	}
}
