package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	valid := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	for value, want := range valid {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			got, err := ParseLevel(value)
			if err != nil {
				t.Fatalf("ParseLevel() = %v", err)
			}
			if got != want {
				t.Fatalf("ParseLevel() = %v, want %v", got, want)
			}
		})
	}

	for _, value := range []string{"", "warning", "INFO", "unknown"} {
		t.Run("invalid_"+value, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseLevel(value); err == nil {
				t.Fatalf("ParseLevel(%q) returned nil error", value)
			}
		})
	}
}

func TestNewWritesStructuredRecordsAtConfiguredLevel(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := New(&output, slog.LevelInfo)
	logger.Debug("hidden")
	logger.Info("ready", "component", "api")

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v", err)
	}
	if got := record["level"]; got != "INFO" {
		t.Errorf("level = %v, want INFO", got)
	}
	if got := record["msg"]; got != "ready" {
		t.Errorf("msg = %v, want ready", got)
	}
	if got := record["component"]; got != "api" {
		t.Errorf("component = %v, want api", got)
	}
}
