// Package testbloby provides file-backed storage for application integration tests.
package testbloby

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/goodies/bloby"
	blobydb "github.com/woodleighschool/goodies/bloby/pgxstore"
)

// New creates an isolated file backend using the supplied object registry.
func New(t *testing.T, db *pgxpool.Pool) *bloby.Service {
	t.Helper()
	service, err := bloby.New(t.Context(), blobydb.New(db), bloby.Config{
		Kind:        bloby.KindFile,
		TransferTTL: time.Minute,
		File: bloby.FileConfig{
			Root:             t.TempDir(),
			BaseURL:          "https://woodstar.example",
			CapabilityKeyHex: strings.Repeat("42", 32),
		},
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	return service
}
