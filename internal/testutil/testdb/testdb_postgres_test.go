//go:build postgres

package testdb

import (
	"net/url"
	"os"
	"testing"
)

func TestOpenPreservesPoolSettings(t *testing.T) {
	baseURL := os.Getenv(testDatabaseURL)
	if baseURL == "" {
		t.Fatalf("%s is required for database tests", testDatabaseURL)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse %s: %v", testDatabaseURL, err)
	}
	query := parsed.Query()
	query.Set("pool_max_conns", "12")
	query.Set("pool_min_conns", "2")
	parsed.RawQuery = query.Encode()
	t.Setenv(testDatabaseURL, parsed.String())

	db, _ := Open(t)
	config := db.Pool().Config()
	if config.MaxConns != 12 || config.MinConns != 2 {
		t.Fatalf("pool bounds = %d/%d, want 12/2", config.MaxConns, config.MinConns)
	}
}
