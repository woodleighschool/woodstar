package config

import "testing"

func TestNormalizeClientIPValidatesEachSource(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{name: "remote_addr", cfg: Config{ClientIPSource: "remote_addr"}},
		{name: "unknown source", cfg: Config{ClientIPSource: "bogus"}, wantErr: true},
		{
			name: "xff cidrs present",
			cfg:  Config{ClientIPSource: "xff_trusted_cidrs", ClientIPTrustedCIDRs: []string{" 10.0.0.0/8 "}},
		},
		{name: "xff cidrs missing", cfg: Config{ClientIPSource: "xff_trusted_cidrs"}, wantErr: true},
		{
			name:    "xff cidrs invalid",
			cfg:     Config{ClientIPSource: "xff_trusted_cidrs", ClientIPTrustedCIDRs: []string{"10.0.0.0"}},
			wantErr: true,
		},
		{
			name: "proxy count",
			cfg:  Config{ClientIPSource: "xff_trusted_proxies", ClientIPTrustedProxies: 2},
		},
		{name: "proxy count zero", cfg: Config{ClientIPSource: "xff_trusted_proxies"}, wantErr: true},
		{name: "header present", cfg: Config{ClientIPSource: "header", ClientIPHeader: "CF-Connecting-IP"}},
		{name: "header missing", cfg: Config{ClientIPSource: "header"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := tc.cfg
			err := cfg.normalizeClientIP()
			if tc.wantErr && err == nil {
				t.Fatal("normalizeClientIP returned nil error, want error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("normalizeClientIP returned error: %v", err)
			}
		})
	}
}

func TestNormalizeClientIPTrimsTrustedCIDRs(t *testing.T) {
	t.Parallel()
	cfg := Config{ClientIPSource: "xff_trusted_cidrs", ClientIPTrustedCIDRs: []string{" 10.0.0.0/8 "}}
	if err := cfg.normalizeClientIP(); err != nil {
		t.Fatalf("normalizeClientIP: %v", err)
	}
	if cfg.ClientIPTrustedCIDRs[0] != "10.0.0.0/8" {
		t.Fatalf("trusted CIDR = %q, want trimmed", cfg.ClientIPTrustedCIDRs[0])
	}
}
