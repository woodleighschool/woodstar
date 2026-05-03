package config

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger configures the process logger with build metadata.
func InitLogger(version string) {
	//nolint:reassign // Zerolog exposes process-wide logger knobs for this setup path.
	zerolog.TimeFieldFormat = time.RFC3339Nano
	//nolint:reassign // The application logger is intentionally configured once at startup.
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}).With().
		Str("version", version).
		Logger()
	ConfigureLogger("info")
}

// ConfigureLogger sets the global zerolog level.
func ConfigureLogger(level string) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn", "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
