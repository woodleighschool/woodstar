package config

import (
	"strings"
	"testing"
)

func TestApplyEnvironmentDefaultsRequireSessionSecret(t *testing.T) {
	t.Setenv("WOODSTAR_HOST", "")
	t.Setenv("WOODSTAR_PORT", "")
	t.Setenv("WOODSTAR_BASE_URL", "")
	t.Setenv("WOODSTAR_SESSION_SECRET", "")
	t.Setenv("WOODSTAR_DATABASE_URL", "")
	t.Setenv("WOODSTAR_LOG_LEVEL", "")

	cfg := Config{}

	err := ApplyEnvironment(&cfg)
	if err == nil {
		t.Fatal("ApplyEnvironment returned nil error, want required session secret error")
	}
}

func TestApplyEnvironmentDefaults(t *testing.T) {
	t.Setenv("WOODSTAR_HOST", "")
	t.Setenv("WOODSTAR_PORT", "")
	t.Setenv("WOODSTAR_BASE_URL", "")
	t.Setenv("WOODSTAR_SESSION_SECRET", strings.Repeat("s", minSessionSecretLength))
	t.Setenv("WOODSTAR_DATABASE_URL", "")
	t.Setenv("WOODSTAR_LOG_LEVEL", "")

	cfg := Config{}

	if err := ApplyEnvironment(&cfg); err != nil {
		t.Fatalf("ApplyEnvironment returned error: %v", err)
	}

	if cfg.Host != "0.0.0.0" {
		t.Fatalf("Host = %q, want 0.0.0.0", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Fatalf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.BaseURL != "http://localhost:8080" {
		t.Fatalf("BaseURL = %q, want http://localhost:8080", cfg.BaseURL)
	}
	if cfg.BasePath() != "/" {
		t.Fatalf("BasePath = %q, want /", cfg.BasePath())
	}
	if len(cfg.SessionSecret) < minSessionSecretLength {
		t.Fatalf("SessionSecret length = %d, want at least %d", len(cfg.SessionSecret), minSessionSecretLength)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want info", cfg.LogLevel)
	}
}

func TestApplyEnvironmentReadsAndNormalizesConfiguredValues(t *testing.T) {
	t.Setenv("WOODSTAR_HOST", "127.0.0.1")
	t.Setenv("WOODSTAR_PORT", "9090")
	t.Setenv("WOODSTAR_BASE_URL", "https://Woodstar.example.edu/woodstar/")
	t.Setenv("WOODSTAR_SESSION_SECRET", strings.Repeat("s", minSessionSecretLength))
	t.Setenv("WOODSTAR_DATABASE_URL", "postgres://example")
	t.Setenv("WOODSTAR_LOG_LEVEL", "debug")

	cfg := Config{}

	err := ApplyEnvironment(&cfg)
	if err != nil {
		t.Fatalf("ApplyEnvironment returned error: %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("Host = %q", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port = %d", cfg.Port)
	}
	if cfg.BaseURL != "https://Woodstar.example.edu/woodstar" {
		t.Fatalf("BaseURL = %q, want https://Woodstar.example.edu/woodstar", cfg.BaseURL)
	}
	if cfg.BasePath() != "/woodstar" {
		t.Fatalf("BasePath = %q, want /woodstar", cfg.BasePath())
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestApplyEnvironmentRejectsWeakSessionSecret(t *testing.T) {
	t.Setenv("WOODSTAR_SESSION_SECRET", "too-short")

	err := ApplyEnvironment(&Config{})
	if err == nil {
		t.Fatal("ApplyEnvironment returned nil error, want weak session secret error")
	}
}
