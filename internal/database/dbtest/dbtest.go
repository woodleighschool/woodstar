package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/database"
)

const testDatabaseURL = "WOODSTAR_TEST_DATABASE_URL"

// Open returns an isolated migrated test database.
func Open(t testing.TB) (*database.DB, context.Context) {
	t.Helper()

	ctx := t.Context()
	baseURL := os.Getenv(testDatabaseURL)
	if baseURL == "" {
		t.Skipf("%s is required for database tests", testDatabaseURL)
	}
	databaseURL := createDatabase(t, ctx, baseURL)

	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(db.Close)

	return db, ctx
}

func createDatabase(t testing.TB, ctx context.Context, baseURL string) string {
	t.Helper()

	databaseName := randomDatabaseName(t)
	adminURL, databaseURL, err := databaseURLs(baseURL, databaseName)
	if err != nil {
		t.Fatalf("parse %s: %v", testDatabaseURL, err)
	}
	admin, err := connectAdmin(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect to test database server: %v", err)
	}
	defer func() { _ = admin.Close(ctx) }()
	identifier := pgx.Identifier{databaseName}.Sanitize()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		cleanupAdmin, err := connectAdmin(cleanupCtx, adminURL)
		if err != nil {
			t.Logf("connect to drop test database %s: %v", databaseName, err)
			return
		}
		defer func() { _ = cleanupAdmin.Close(cleanupCtx) }()
		_, err = cleanupAdmin.Exec(cleanupCtx, "DROP DATABASE IF EXISTS "+identifier+" WITH (FORCE)")
		if err != nil {
			t.Logf("drop test database %s: %v", databaseName, err)
		}
	})
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		t.Fatalf("create test database %s: %v", databaseName, err)
	}
	return databaseURL
}

func connectAdmin(ctx context.Context, databaseURL string) (*pgx.Conn, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	return pgx.ConnectConfig(ctx, config.ConnConfig)
}

func databaseURLs(baseURL string, databaseName string) (string, string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", "", err
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", "", fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	admin := *parsed
	admin.Path = "/postgres"
	target := *parsed
	target.Path = "/" + databaseName
	return admin.String(), target.String(), nil
}

func randomDatabaseName(t testing.TB) string {
	t.Helper()

	var entropy [8]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatalf("create random test database name: %v", err)
	}
	return "woodstar_test_" + hex.EncodeToString(entropy[:])
}
