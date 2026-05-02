package config

import "os"

type Config struct {
	Host        string
	Port        int
	DatabaseURL string
	LogLevel    string
}

func ApplyEnvironment(cfg *Config) {
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = os.Getenv("WOODSTAR_DATABASE_URL")
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = os.Getenv("WOODSTAR_LOG_LEVEL")
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
}
