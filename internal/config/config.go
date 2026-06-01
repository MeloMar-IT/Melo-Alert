package config

import (
	"fmt"
	"os"
	"time"

	"signalhub/internal/domain"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Routing  domain.RoutingConfig `yaml:"routing"`
	Teams    TeamsConfig    `yaml:"teams"`
	ServiceNow ServiceNowConfig `yaml:"servicenow"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServiceNowConfig struct {
	Enabled      bool          `yaml:"enabled"`
	InstanceURL  string        `yaml:"instance_url"`
	User         string        `yaml:"user"`
	Password     string        `yaml:"password"`
	Timeout      time.Duration `yaml:"timeout"`
	AssignmentGroup string     `yaml:"assignment_group"`
}

type TeamsConfig struct {
	Enabled    bool          `yaml:"enabled"`
	WebhookURL string        `yaml:"webhook_url"`
	Timeout    time.Duration `yaml:"timeout"`
}

type ServerConfig struct {
	Address      string        `yaml:"address"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type AuthConfig struct {
	WebhookToken string `yaml:"webhook_token"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}

	// Default values
	cfg.Server.Address = ":8080"
	cfg.Server.ReadTimeout = 5 * time.Second
	cfg.Server.WriteTimeout = 10 * time.Second
	cfg.Teams.Timeout = 10 * time.Second
	cfg.ServiceNow.Timeout = 10 * time.Second
	cfg.Logging.Level = "info"

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// Expand environment variables in the YAML file
		expandedData := os.ExpandEnv(string(data))

		if err := yaml.Unmarshal([]byte(expandedData), cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	return cfg, nil
}
