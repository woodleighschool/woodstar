package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/munki/hoststate"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
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
	if got.Agents.Osquery.DistributedIntervalSeconds == nil ||
		*got.Agents.Osquery.DistributedIntervalSeconds != 15 {
		t.Fatalf("distributed interval seconds = %v, want 15", got.Agents.Osquery.DistributedIntervalSeconds)
	}
	if got.Agents.Osquery.ConfigRefreshSeconds == nil ||
		*got.Agents.Osquery.ConfigRefreshSeconds != 60 {
		t.Fatalf("config refresh seconds = %v, want 60", got.Agents.Osquery.ConfigRefreshSeconds)
	}

	got = parseOsqueryFlags([]map[string]string{{"name": "config_tls_refresh", "value": "30"}})
	if got.Agents.Osquery.ConfigRefreshSeconds == nil ||
		*got.Agents.Osquery.ConfigRefreshSeconds != 30 {
		t.Fatalf("config refresh seconds = %v, want 30", got.Agents.Osquery.ConfigRefreshSeconds)
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

func TestIngestMunkiDetailRows(t *testing.T) {
	store := &fakeMunkiStore{}
	projector := (&Projector{}).WithMunkiStore(store)
	ctx := context.Background()

	if err := projector.IngestDetail(
		ctx,
		catalog.DetailQueries()[catalog.QueryMunkiInfo],
		catalog.QueryMunkiInfo,
		42,
		[]map[string]string{{
			"version":       "7.1.2.5700",
			"manifest_name": "site_default",
			"success":       "true",
			"errors":        "first; second",
		}},
	); err != nil {
		t.Fatalf("ingest munki info: %v", err)
	}
	if store.status.HostID != 42 || store.status.Version != "7.1.2.5700" ||
		store.status.ManifestName != "site_default" {
		t.Fatalf("status = %+v, want parsed munki info", store.status)
	}
	if len(store.status.Errors) != 2 {
		t.Fatalf("errors = %#v, want two entries", store.status.Errors)
	}

	if err := projector.IngestDetail(
		ctx,
		catalog.DetailQueries()[catalog.QueryMunkiInstalls],
		catalog.QueryMunkiInstalls,
		42,
		[]map[string]string{{
			"name":              "GoogleChrome",
			"installed":         "true",
			"installed_version": "148.0",
		}},
	); err != nil {
		t.Fatalf("ingest munki installs: %v", err)
	}
	if len(store.items) != 1 || store.items[0].Name != "GoogleChrome" || !store.items[0].Installed {
		t.Fatalf("items = %+v, want parsed munki item", store.items)
	}

	if err := projector.IngestDetail(
		ctx,
		catalog.DetailQueries()[catalog.QueryMunkiInfo],
		catalog.QueryMunkiInfo,
		42,
		nil,
	); err != nil {
		t.Fatalf("ingest missing munki info: %v", err)
	}
	if store.clearedHostID != 42 {
		t.Fatalf("clearedHostID = %d, want 42", store.clearedHostID)
	}
}

type fakeMunkiStore struct {
	status        hoststate.Observation
	items         []hoststate.Item
	clearedHostID int64
}

func (s *fakeMunkiStore) UpsertHostStatus(_ context.Context, status hoststate.Observation) error {
	s.status = status
	return nil
}

func (s *fakeMunkiStore) ClearHostStatus(_ context.Context, hostID int64) error {
	s.clearedHostID = hostID
	return nil
}

func (s *fakeMunkiStore) ReplaceHostItems(_ context.Context, _ int64, items []hoststate.Item) error {
	s.items = items
	return nil
}
