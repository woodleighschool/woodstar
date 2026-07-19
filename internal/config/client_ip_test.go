package config

import (
	"strings"
	"testing"
	"time"
)

func TestConfigValidatesEachClientIPSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		update  func(*Config)
		wantErr bool
	}{
		{name: "remote address", update: func(*Config) {}},
		{name: "unknown source", update: func(cfg *Config) { cfg.ClientIPSource = "bogus" }, wantErr: true},
		{
			name: "trusted CIDRs",
			update: func(cfg *Config) {
				cfg.ClientIPSource = ClientIPSourceXFFTrustedCIDRs
				cfg.ClientIPTrustedCIDRs = []string{" 10.0.0.0/8 "}
			},
		},
		{
			name:    "missing trusted CIDRs",
			update:  func(cfg *Config) { cfg.ClientIPSource = ClientIPSourceXFFTrustedCIDRs },
			wantErr: true,
		},
		{
			name: "invalid trusted CIDR",
			update: func(cfg *Config) {
				cfg.ClientIPSource = ClientIPSourceXFFTrustedCIDRs
				cfg.ClientIPTrustedCIDRs = []string{"10.0.0.0"}
			},
			wantErr: true,
		},
		{
			name: "trusted proxy count",
			update: func(cfg *Config) {
				cfg.ClientIPSource = ClientIPSourceXFFTrustedProxies
				cfg.ClientIPTrustedProxies = 2
			},
		},
		{
			name:    "missing trusted proxy count",
			update:  func(cfg *Config) { cfg.ClientIPSource = ClientIPSourceXFFTrustedProxies },
			wantErr: true,
		},
		{
			name: "trusted header",
			update: func(cfg *Config) {
				cfg.ClientIPSource = ClientIPSourceHeader
				cfg.ClientIPHeader = "CF-Connecting-IP"
			},
		},
		{
			name:    "missing trusted header",
			update:  func(cfg *Config) { cfg.ClientIPSource = ClientIPSourceHeader },
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg := validConfig()
			test.update(&cfg)
			cfg.Normalize()
			err := cfg.Validate()
			if test.wantErr && err == nil {
				t.Fatal("validate returned nil error, want error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("validate: %v", err)
			}
		})
	}
}

func TestConfigNormalizesTrustedCIDRs(t *testing.T) {
	t.Parallel()
	cfg := validConfig()
	cfg.ClientIPSource = ClientIPSourceXFFTrustedCIDRs
	cfg.ClientIPTrustedCIDRs = []string{" 10.0.0.0/8 "}

	cfg.Normalize()

	if cfg.ClientIPTrustedCIDRs[0] != "10.0.0.0/8" {
		t.Fatalf("trusted CIDR = %q, want trimmed", cfg.ClientIPTrustedCIDRs[0])
	}
}

func validConfig() Config {
	return Config{
		Host:                    "0.0.0.0",
		Port:                    8080,
		ServerURL:               "https://localhost:8080",
		StorageCapabilityKey:    strings.Repeat("a", storageCapabilityKeyHexLength),
		SessionCookieSecure:     true,
		DatabaseURL:             "postgres://woodstar:woodstar@localhost:5432/woodstar",
		LogLevel:                "info",
		SantaEventRetentionDays: 90,
		SantaEventSweepInterval: time.Hour,
		OIDCScopes:              []string{"openid", "email", "profile"},
		OIDCEmailClaim:          "email",
		OIDCRedirectURL:         "https://localhost:8080/api/auth/sso/callback",
		EntraSyncInterval:       time.Hour,
		StorageKind:             "file",
		StorageFileRoot:         "data/storage",
		StorageTransferTTL:      15 * time.Minute,
		ClientIPSource:          ClientIPSourceRemoteAddr,
	}
}
