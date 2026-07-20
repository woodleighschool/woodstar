//go:build postgres

package testdb

import (
	"context"
	"os"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
)

const testDatabaseURL = "WOODSTAR_TEST_DATABASE_URL"

// Open returns an isolated migrated test database.
func Open(t testing.TB) (*database.DB, context.Context) {
	t.Helper()

	ctx := t.Context()
	baseURL := os.Getenv(testDatabaseURL)
	if baseURL == "" {
		t.Fatalf("%s is required for database tests", testDatabaseURL)
	}
	databaseURL := Create(t, baseURL)

	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(db.Close)

	return db, ctx
}
