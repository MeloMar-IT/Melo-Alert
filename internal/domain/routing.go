package domain

import (
	"regexp"
)

type Route struct {
	Receiver string   `yaml:"receiver"`
	Match    Match    `yaml:"match,omitempty"`
	MatchRE  Match    `yaml:"match_re,omitempty"`
	Routes   []Route  `yaml:"routes,omitempty"`
	Continue bool     `yaml:"continue,omitempty"`
}

type Match map[string]string

type RoutingConfig struct {
	DefaultReceiver string  `yaml:"default_receiver"`
	Routes          []Route `yaml:"routes"`
}

type Matcher struct {
	config RoutingConfig
}

func NewMatcher(cfg RoutingConfig) *Matcher {
	return &Matcher{config: cfg}
}

func (m *Matcher) Match(alert *Alert) []string {
	var receivers []string
	
	matched := false
	for _, route := range m.config.Routes {
		if r, ok := m.matchRoute(alert, &route); ok {
			receivers = append(receivers, r...)
			matched = true
			if !route.Continue {
				break
			}
		}
	}

	if !matched && m.config.DefaultReceiver != "" {
		receivers = append(receivers, m.config.DefaultReceiver)
	}

	return receivers
}

func (m *Matcher) matchRoute(alert *Alert, route *Route) ([]string, bool) {
	// Check if this route matches
	if !m.matches(alert, route.Match, false) {
		return nil, false
	}
	if !m.matches(alert, route.MatchRE, true) {
		return nil, false
	}

	var receivers []string
	
	// Check sub-routes
	subMatched := false
	for _, subRoute := range route.Routes {
		if r, ok := m.matchRoute(alert, &subRoute); ok {
			receivers = append(receivers, r...)
			subMatched = true
			if !subRoute.Continue {
				break
			}
		}
	}

	// If no sub-routes matched, but this route matches, use this route's receiver
	if !subMatched && route.Receiver != "" {
		receivers = append(receivers, route.Receiver)
		return receivers, true
	}

	if subMatched {
		return receivers, true
	}

	// If this route matched (it has no sub-routes or sub-routes didn't match but we match), 
	// but it has no receiver, it's not a full match unless we want to allow intermediate nodes.
	// In Prometheus Alertmanager, a route without a receiver is valid if it has subroutes.
	return nil, false
}

func (m *Matcher) matches(alert *Alert, match Match, isRegex bool) bool {
	for field, value := range match {
		var alertValue string
		switch field {
		case "severity":
			alertValue = alert.Severity
		case "environment":
			alertValue = alert.Environment
		case "service":
			alertValue = alert.Service
		case "team":
			alertValue = alert.Team
		case "source":
			alertValue = alert.Source
		case "resource":
			alertValue = alert.Resource
		case "status":
			alertValue = string(alert.Status)
		default:
			// Check labels
			if val, ok := alert.Labels[field]; ok {
				alertValue = val
			} else {
				return false
			}
		}

		if isRegex {
			matched, _ := regexp.MatchString("^"+value+"$", alertValue)
			if !matched {
				return false
			}
		} else {
			if alertValue != value {
				return false
			}
		}
	}
	return true
}
