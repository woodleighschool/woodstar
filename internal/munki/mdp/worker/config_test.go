package worker

import (
	"errors"
	"strings"
	"testing"

	"github.com/caarlos0/env/v11"
)

func setRequiredConfigEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("WOODSTAR_MDP_SERVER_URL", "https://woodstar.example/")
	t.Setenv("WOODSTAR_MDP_KEY", "distribution-point-key")
	t.Setenv("WOODSTAR_MDP_DATA_DIR", t.TempDir())
	t.Setenv("WOODSTAR_MDP_LISTEN_ADDR", "")
	t.Setenv("WOODSTAR_MDP_LOG_LEVEL", "")
	t.Setenv("WOODSTAR_MDP_DOWNLOAD_CONCURRENCY", "")
	t.Setenv("WOODSTAR_MDP_TLS_CERT_FILE", "")
	t.Setenv("WOODSTAR_MDP_TLS_KEY_FILE", "")
}

func TestLoadConfigDefaultsAndNormalizes(t *testing.T) {
	setRequiredConfigEnvironment(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.ServerURL != "https://woodstar.example" {
		t.Fatalf("ServerURL = %q, want normalized origin", cfg.ServerURL)
	}
	if cfg.ListenAddr != ":8080" {
		t.Fatalf("ListenAddr = %q, want :8080", cfg.ListenAddr)
	}
	if cfg.DownloadConcurrency != 4 {
		t.Fatalf("DownloadConcurrency = %d, want 4", cfg.DownloadConcurrency)
	}
	if cfg.TLSConfigured() {
		t.Fatal("TLSConfigured = true without certificate and key files")
	}
}

func TestLoadConfigReturnsEnvironmentInputErrors(t *testing.T) {
	setRequiredConfigEnvironment(t)
	t.Setenv("WOODSTAR_MDP_SERVER_URL", "")
	t.Setenv("WOODSTAR_MDP_KEY", "")
	t.Setenv("WOODSTAR_MDP_DATA_DIR", "")

	_, err := LoadConfig()
	var aggregate env.AggregateError
	if !errors.As(err, &aggregate) {
		t.Fatalf("LoadConfig error = %T, want env.AggregateError", err)
	}
	if len(aggregate.Errors) != 3 {
		t.Fatalf("aggregate errors = %d, want 3", len(aggregate.Errors))
	}
	for _, name := range []string{
		"WOODSTAR_MDP_SERVER_URL",
		"WOODSTAR_MDP_KEY",
		"WOODSTAR_MDP_DATA_DIR",
	} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("LoadConfig error %q does not name %s", err, name)
		}
	}
}

func TestLoadConfigReadsTLSFiles(t *testing.T) {
	setRequiredConfigEnvironment(t)
	t.Setenv("WOODSTAR_MDP_TLS_CERT_FILE", " /etc/woodstar/mdp.crt ")
	t.Setenv("WOODSTAR_MDP_TLS_KEY_FILE", " /etc/woodstar/mdp.key ")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.TLSCertFile != "/etc/woodstar/mdp.crt" || cfg.TLSKeyFile != "/etc/woodstar/mdp.key" {
		t.Fatalf("TLS files = %q, %q", cfg.TLSCertFile, cfg.TLSKeyFile)
	}
	if !cfg.TLSConfigured() {
		t.Fatal("TLSConfigured = false with certificate and key files")
	}
}

func TestLoadConfigRejectsHTTPServerURL(t *testing.T) {
	setRequiredConfigEnvironment(t)
	t.Setenv("WOODSTAR_MDP_SERVER_URL", "http://woodstar.example")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig returned nil error, want HTTP server URL rejection")
	}
}

func TestLoadConfigRejectsServerURLPath(t *testing.T) {
	setRequiredConfigEnvironment(t)
	t.Setenv("WOODSTAR_MDP_SERVER_URL", "https://woodstar.example/base")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig returned nil error, want server URL path rejection")
	}
}

func TestLoadConfigRejectsPartialTLSConfig(t *testing.T) {
	setRequiredConfigEnvironment(t)
	t.Setenv("WOODSTAR_MDP_TLS_CERT_FILE", "/etc/woodstar/mdp.crt")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig returned nil error, want partial TLS configuration rejection")
	}
}

func TestLoadConfigRejectsNonPositiveDownloadConcurrency(t *testing.T) {
	setRequiredConfigEnvironment(t)
	t.Setenv("WOODSTAR_MDP_DOWNLOAD_CONCURRENCY", "0")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig returned nil error, want download concurrency rejection")
	}
}

func TestLoadConfigRejectsInvalidLogLevel(t *testing.T) {
	setRequiredConfigEnvironment(t)
	t.Setenv("WOODSTAR_MDP_LOG_LEVEL", "warning")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig returned nil error, want log level rejection")
	}
}
