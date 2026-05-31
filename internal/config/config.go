package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Logging  LoggingConfig  `yaml:"logging"`
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
