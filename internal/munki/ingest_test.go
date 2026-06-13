package munki_test

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/munki"
)

func TestDetailIngestorProjectsRows(t *testing.T) {
	store := &fakeHostStateStore{}
	ingestor := munki.NewDetailIngestor(store)
	ctx := context.Background()

	if err := ingestor.IngestInfo(ctx, 42, []map[string]string{{
		"version":       "7.1.2.5700",
		"manifest_name": "site_default",
		"success":       "true",
		"errors":        "first; second",
	}}); err != nil {
		t.Fatalf("ingest munki info: %v", err)
	}
	if store.status.HostID != 42 || store.status.Version != "7.1.2.5700" ||
		store.status.ManifestName != "site_default" {
		t.Fatalf("status = %+v, want parsed munki info", store.status)
	}
	if len(store.status.Errors) != 2 {
		t.Fatalf("errors = %#v, want two entries", store.status.Errors)
	}

	if err := ingestor.IngestInstalls(ctx, 42, []map[string]string{{
		"name":              "GoogleChrome",
		"installed":         "true",
		"installed_version": "148.0",
	}}); err != nil {
		t.Fatalf("ingest munki installs: %v", err)
	}
	if len(store.items) != 1 || store.items[0].Name != "GoogleChrome" || !store.items[0].Installed {
		t.Fatalf("items = %+v, want parsed munki item", store.items)
	}

	if err := ingestor.IngestInfo(ctx, 42, nil); err != nil {
		t.Fatalf("ingest missing munki info: %v", err)
	}
	if store.clearedHostID != 42 {
		t.Fatalf("clearedHostID = %d, want 42", store.clearedHostID)
	}
}

type fakeHostStateStore struct {
	status        munki.HostObservation
	items         []munki.Item
	clearedHostID int64
}

func (s *fakeHostStateStore) UpsertHostObservation(_ context.Context, status munki.HostObservation) error {
	s.status = status
	return nil
}

func (s *fakeHostStateStore) ClearHostObservation(_ context.Context, hostID int64) error {
	s.clearedHostID = hostID
	return nil
}

func (s *fakeHostStateStore) ReplaceHostItems(_ context.Context, _ int64, items []munki.Item) error {
	s.items = items
	return nil
}
