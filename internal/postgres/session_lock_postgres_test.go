//go:build postgres

package postgres_test

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/postgres"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestSessionLockerLeavesThePoolAvailableToWork(t *testing.T) {
	baseURL := os.Getenv("WOODSTAR_TEST_DATABASE_URL")
	if baseURL == "" {
		t.Fatal("WOODSTAR_TEST_DATABASE_URL is required for database tests")
	}
	databaseURL := testdb.Create(t, baseURL)
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	query := parsed.Query()
	query.Set("pool_max_conns", "1")
	parsed.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, parsed.String())
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	locker := postgres.NewSessionLocker(pool, 7146808627076917999)
	acquired, err := locker.Try(ctx, func(ctx context.Context) error {
		return pool.Ping(ctx)
	})
	if err != nil {
		t.Fatalf("run locked work: %v", err)
	}
	if !acquired {
		t.Fatal("advisory lock was unexpectedly busy")
	}
}
