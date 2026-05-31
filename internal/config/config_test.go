package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		cfg, err := Load("")
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if cfg.Server.Address != ":8080" {
			t.Errorf("expected :8080, got %s", cfg.Server.Address)
		}
		if cfg.Server.ReadTimeout != 5*time.Second {
			t.Errorf("expected 5s, got %v", cfg.Server.ReadTimeout)
		}
	})

	t.Run("env expansion", func(t *testing.T) {
		os.Setenv("TEST_DSN", "postgres://localhost:5432")
		defer os.Unsetenv("TEST_DSN")

		yamlContent := `
database:
  dsn: "${TEST_DSN}"
`
		tmpFile, err := os.CreateTemp("", "config*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write([]byte(yamlContent)); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		cfg, err := Load(tmpFile.Name())
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if cfg.Database.DSN != "postgres://localhost:5432" {
			t.Errorf("expected postgres://localhost:5432, got %s", cfg.Database.DSN)
		}
	})
}
