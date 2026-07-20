package testdb

import (
	"net/url"
	"testing"
)

func TestDatabaseURLsPreserveConnectionSettings(t *testing.T) {
	t.Parallel()

	const base = "postgres://woodstar:secret@database.test:5432/custom?sslmode=disable&pool_max_conns=12" //nolint:gosec // Fixed test database credential.
	adminURL, databaseURL, err := databaseURLs(base, "woodstar_test_example")
	if err != nil {
		t.Fatalf("databaseURLs: %v", err)
	}

	for name, rawURL := range map[string]string{"admin": adminURL, "database": databaseURL} {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			t.Fatalf("parse %s URL: %v", name, err)
		}
		if parsed.User.String() != "woodstar:secret" ||
			parsed.Host != "database.test:5432" ||
			parsed.Query().Get("sslmode") != "disable" ||
			parsed.Query().Get("pool_max_conns") != "12" {
			t.Fatalf("%s URL lost connection settings: %s", name, rawURL)
		}
	}
	if parsed, _ := url.Parse(adminURL); parsed.Path != "/postgres" {
		t.Fatalf("admin database path = %q, want /postgres", parsed.Path)
	}
	if parsed, _ := url.Parse(databaseURL); parsed.Path != "/woodstar_test_example" {
		t.Fatalf("isolated database path = %q, want /woodstar_test_example", parsed.Path)
	}
}

func TestDatabaseURLsRejectUnsupportedScheme(t *testing.T) {
	t.Parallel()

	if _, _, err := databaseURLs("mysql://database.test/woodstar", "woodstar_test_example"); err == nil {
		t.Fatal("databaseURLs accepted a non-PostgreSQL URL")
	}
}
