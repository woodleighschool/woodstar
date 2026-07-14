package logging

import (
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
