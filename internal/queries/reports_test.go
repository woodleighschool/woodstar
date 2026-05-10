package queries

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSnapshotResultRowsStoreEachOsqueryRowSeparately(t *testing.T) {
	fetchedAt := time.Date(2026, 5, 9, 10, 30, 0, 0, time.UTC)
	rows, err := snapshotResultRows([]map[string]string{
		{"name": "alpha", "version": "1"},
		{"name": "bravo", "version": "2"},
	}, fetchedAt)
	if err != nil {
		t.Fatalf("snapshotResultRows returned error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	for i, row := range rows {
		if row.data == nil {
			t.Fatalf("rows[%d].data is nil, want JSON object", i)
		}
		var got map[string]string
		if err := json.Unmarshal(*row.data, &got); err != nil {
			t.Fatalf("unmarshal rows[%d]: %v", i, err)
		}
		if got["name"] == "" || got["version"] == "" {
			t.Fatalf("rows[%d] data = %#v, want osquery columns", i, got)
		}
		if !row.lastFetched.Equal(fetchedAt) {
			t.Fatalf("rows[%d].lastFetched = %s, want %s", i, row.lastFetched, fetchedAt)
		}
	}
}

func TestSnapshotResultRowsPreserveEmptyFetchWithNullData(t *testing.T) {
	fetchedAt := time.Date(2026, 5, 9, 10, 30, 0, 0, time.UTC)
	rows, err := snapshotResultRows(nil, fetchedAt)
	if err != nil {
		t.Fatalf("snapshotResultRows returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].data != nil {
		t.Fatalf("rows[0].data = %s, want nil sentinel", string(*rows[0].data))
	}
	if !rows[0].lastFetched.Equal(fetchedAt) {
		t.Fatalf("rows[0].lastFetched = %s, want %s", rows[0].lastFetched, fetchedAt)
	}
}
