package ingest

import (
	"testing"
	"time"
)

func TestParseSoftwareRows(t *testing.T) {
	rows := []map[string]string{
		{
			"name":              "Safari",
			"version":           "26.0",
			"source":            "apps",
			"bundle_identifier": "com.apple.Safari",
			"installed_path":    "/Applications/Safari.app",
			"last_opened_at":    "1745999192.82046",
		},
		{
			"name":              "node",
			"version":           "24.0.0",
			"source":            "homebrew_packages",
			"last_opened_at":    "",
			"bundle_identifier": "",
		},
		{"name": ""},
	}

	got := parseSoftwareRows(rows, softwareEnrichment{})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "Safari" || got[0].Source != "apps" || got[0].InstalledPath != "/Applications/Safari.app" {
		t.Fatalf("first row parsed incorrectly: %#v", got[0])
	}
	wantOpened := time.Unix(1745999192, 820460000).UTC()
	if got[0].LastOpenedAt == nil || !got[0].LastOpenedAt.Equal(wantOpened) {
		t.Fatalf("LastOpenedAt = %v, want %v", got[0].LastOpenedAt, wantOpened)
	}
	if got[1].LastOpenedAt != nil {
		t.Fatalf("second LastOpenedAt = %v, want nil", got[1].LastOpenedAt)
	}
}

func TestParseOsqueryFlags(t *testing.T) {
	got := parseOsqueryFlags([]map[string]string{
		{"name": "distributed_interval", "value": "15"},
		{"name": "config_tls_refresh", "value": "60"},
	})
	if got.DistributedInterval == nil || *got.DistributedInterval != 15 {
		t.Fatalf("DistributedInterval = %v, want 15", got.DistributedInterval)
	}
	if got.ConfigTLSRefresh == nil || *got.ConfigTLSRefresh != 60 {
		t.Fatalf("ConfigTLSRefresh = %v, want 60", got.ConfigTLSRefresh)
	}

	got = parseOsqueryFlags([]map[string]string{{"name": "config_tls_refresh", "value": "30"}})
	if got.ConfigTLSRefresh == nil || *got.ConfigTLSRefresh != 30 {
		t.Fatalf("ConfigTLSRefresh = %v, want 30", got.ConfigTLSRefresh)
	}
}

func TestParseHostCertificatesStructuresDistinguishedNames(t *testing.T) {
	got := parseHostCertificates("certificates_darwin", []map[string]string{{
		"sha1":        "sha1",
		"common_name": "Subject CN",
		"subject":     `/C=AU/O=Example Org/OU=One/OU=Two\/WithSlash/CN=Subject CN`,
		"issuer":      `/C=US/O=Issuer Org/OU=Root/CN=Issuer CN`,
		"path":        "/Users/alice/Library/Keychains/login.keychain-db",
	}})

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	certificate := got[0]
	if certificate.Subject.Country != "AU" ||
		certificate.Subject.Organization != "Example Org" ||
		certificate.Subject.OrganizationalUnit != "One+OU=Two/WithSlash" ||
		certificate.Subject.CommonName != "Subject CN" ||
		certificate.Issuer.CommonName != "Issuer CN" {
		t.Fatalf(
			"certificate names = %#v / %#v, want parsed subject and issuer",
			certificate.Subject,
			certificate.Issuer,
		)
	}
	if certificate.Source != "user" || certificate.Username != "alice" {
		t.Fatalf("source/username = %q/%q, want user/alice", certificate.Source, certificate.Username)
	}
}

func TestParseSoftwareRowsEnrichesInstalledPaths(t *testing.T) {
	rows := []map[string]string{
		{
			"name":              "Example",
			"version":           "1.2.3",
			"source":            "apps",
			"bundle_identifier": "com.example.app",
			"installed_path":    "/Applications/Example.app",
		},
	}
	enrichment := softwareEnrichmentByPath([]map[string]string{{
		"path":            "/Applications/Example.app",
		"team_identifier": "ABCD123456",
		"cdhash_sha256":   "cdhash",
	}}, []map[string]string{{
		"path":              "/Applications/Example.app",
		"executable_sha256": "executable-hash",
		"executable_path":   "/Applications/Example.app/Contents/MacOS/Example",
		"unrelated_ignored": "ignored",
	}})

	got := parseSoftwareRows(rows, enrichment)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].TeamIdentifier != "ABCD123456" || got[0].CDHashSHA256 != "cdhash" ||
		got[0].ExecutableSHA256 != "executable-hash" ||
		got[0].ExecutablePath != "/Applications/Example.app/Contents/MacOS/Example" {
		t.Fatalf("enrichment parsed incorrectly: %#v", got[0])
	}
}
