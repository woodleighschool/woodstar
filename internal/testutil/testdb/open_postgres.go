//go:build postgres

package testdb

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

const testDatabaseURL = "WOODSTAR_TEST_DATABASE_URL"

// Open returns an isolated migrated test database.
func Open(t testing.TB) (*pgxpool.Pool, context.Context) {
	t.Helper()

	ctx := t.Context()
	baseURL := os.Getenv(testDatabaseURL)
	if baseURL == "" {
		t.Fatalf("%s is required for database tests", testDatabaseURL)
	}
	databaseURL := Create(t, baseURL)

	db, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(db.Close)

	return db, ctx
}
