//go:build postgres

package munki_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestHostCollectionReconcilesOnlyChangedItems(t *testing.T) {
	db, ctx := testdb.Open(t)
	collector := munki.NewDetailIngestor(munki.NewStore(db))
	host := collectionHost(t, db, "munki-collection")
	other := collectionHost(t, db, "munki-other")
	collect := func(hostID int64, items ...map[string]string) {
		t.Helper()
		if err := collector.IngestCollection(ctx, hostID, munki.Collection{
			Info:     munki.QueryResult{Present: true, Successful: true, Rows: []map[string]string{{"version": "7.1"}}},
			Installs: munki.QueryResult{Present: true, Successful: true, Rows: items},
		}); err != nil {
			t.Fatal(err)
		}
	}
	type state struct {
		statusXmin, itemXmin, installedVersion string
	}
	read := func(hostID int64) state {
		t.Helper()
		var s state
		err := db.QueryRow(ctx, `
			SELECT status.xmin::text, item.xmin::text, item.installed_version
			FROM munki_host_status status JOIN munki_host_items item USING (host_id)
			WHERE host_id = $1 AND name = 'Editor'`, hostID).Scan(&s.statusXmin, &s.itemXmin, &s.installedVersion)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	editor := map[string]string{"name": "Editor", "installed_version": "2"}
	collect(host, editor, map[string]string{"name": "Removed"})
	collect(other, editor)
	before, otherBefore := read(host), read(other)

	collect(host, map[string]string{"name": "Editor", "installed_version": "1"}, editor)
	if got := read(host); got != before {
		t.Errorf("duplicate ending in the current version changed status or item: before %+v, after %+v", before, got)
	}
	var count int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM munki_host_items WHERE host_id = $1`, host).Scan(&count); err != nil || count != 1 {
		t.Fatalf("remaining item count = %d, %v; want 1", count, err)
	}
	collect(host, editor, map[string]string{"name": "Editor", "installed_version": "3"})
	if got := read(host); got.statusXmin != before.statusXmin || got.installedVersion != "3" {
		t.Errorf("updated snapshot = %+v, want unchanged status and last reported version 3", got)
	}
	collect(host)
	if err := db.QueryRow(ctx, `
		SELECT count(item.name) FROM munki_host_status status LEFT JOIN munki_host_items item USING (host_id)
		WHERE host_id = $1 GROUP BY host_id`, host).Scan(&count); err != nil || count != 0 {
		t.Fatalf("empty report item count = %d, %v; want status and no items", count, err)
	}
	if got := read(other); got != otherBefore {
		t.Fatalf("other host changed: before %+v, after %+v", otherBefore, got)
	}
}

func TestAbsentHostCollectionWaitsForConcurrentReport(t *testing.T) {
	db, ctx := testdb.Open(t)
	collector := munki.NewDetailIngestor(munki.NewStore(db))
	host := collectionHost(t, db, "munki-absent")
	runBlockedPatch(t, db, "hosts", host, func(ctx context.Context) error {
		return collector.IngestCollection(ctx, host, munki.Collection{})
	}, func(ctx context.Context, tx pgx.Tx) {
		if _, err := tx.Exec(ctx, `INSERT INTO munki_host_status (host_id) VALUES ($1)`, host); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO munki_host_items (host_id, name) VALUES ($1, 'Editor')`, host); err != nil {
			t.Fatal(err)
		}
	})
	var present bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS (SELECT FROM munki_host_status WHERE host_id = $1)
			OR EXISTS (SELECT FROM munki_host_items WHERE host_id = $1)`, host).Scan(&present); err != nil || present {
		t.Fatalf("Munki state after absent collection = %t, %v; want no state", present, err)
	}
}

func collectionHost(t *testing.T, db *pgxpool.Pool, uuid string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(t.Context(), `INSERT INTO hosts (hardware_uuid) VALUES ($1) RETURNING id`, uuid).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
