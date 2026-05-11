package dbtest

import (
	"context"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/woodleighschool/woodstar/internal/database"
)

const (
	testDatabaseURL = "WOODSTAR_TEST_DATABASE_URL"
	postgresImage   = "postgres:16-alpine"
)

// Open returns a migrated test database. CI is expected to provide Postgres via
// WOODSTAR_TEST_DATABASE_URL; local tests fall back to testcontainers.
func Open(t *testing.T) (*database.DB, context.Context) {
	t.Helper()

	ctx := context.Background()
	databaseURL := os.Getenv(testDatabaseURL)
	if databaseURL == "" {
		if os.Getenv("CI") != "" {
			t.Skipf("%s is required in CI", testDatabaseURL)
		}
		databaseURL = startPostgres(t, ctx)
	}

	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(db.Close)

	return db, ctx
}

func startPostgres(t *testing.T, ctx context.Context) string {
	t.Helper()

	ctr, err := postgres.Run(
		ctx,
		postgresImage,
		postgres.WithDatabase("woodstar_test"),
		postgres.WithUsername("woodstar"),
		postgres.WithPassword("woodstar"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("start local Postgres test container: %v", err)
	}
	t.Cleanup(func() {
		if err := ctr.Terminate(context.Background()); err != nil {
			t.Logf("terminate Postgres test container: %v", err)
		}
	})

	databaseURL, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Skipf("build Postgres test container URL: %v", err)
	}
	return databaseURL
}
