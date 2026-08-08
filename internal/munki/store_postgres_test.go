//go:build postgres

package munki_test

import (
	"context"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestLoadHostStateDerivesReportState(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := munki.NewStore(db)
	successAt := time.Date(2026, 8, 2, 1, 2, 3, 0, time.UTC)

	fixtures := []struct {
		serial          string
		lastSuccessful  *time.Time
		collectionError string
		hasReport       bool
		want            munki.ReportState
	}{
		{serial: "C02MUNKICURRENT", lastSuccessful: &successAt, hasReport: true, want: munki.ReportCurrent},
		{serial: "C02MUNKINOREPORT", lastSuccessful: &successAt, want: munki.ReportNoReport},
		{serial: "C02MUNKIFAILURE", lastSuccessful: &successAt, collectionError: "collector unavailable", want: munki.ReportCollectionFailed},
		{serial: "C02MUNKIFIRSTFAILURE", collectionError: "collector unavailable", want: munki.ReportNeverCollected},
	}
	for _, fixture := range fixtures {
		host := munkiReportTestHost(t, ctx, db, fixture.serial)
		if _, err := db.Pool().Exec(ctx, `
INSERT INTO munki_host_status (
    host_id, last_attempt_at, last_successful_at, collection_error, has_report
) VALUES ($1, $2, $3, $4, $5)`,
			host.ID, successAt, fixture.lastSuccessful, fixture.collectionError, fixture.hasReport); err != nil {
			t.Fatalf("insert %s status: %v", fixture.serial, err)
		}

		state, err := store.LoadHostState(ctx, host.ID)
		if err != nil {
			t.Fatalf("LoadHostState(%s): %v", fixture.serial, err)
		}
		if state == nil || state.ReportState != fixture.want {
			t.Fatalf("LoadHostState(%s) = %+v, want %s", fixture.serial, state, fixture.want)
		}
		if state.CollectionError != fixture.collectionError || state.LastAttemptAt == nil {
			t.Fatalf("LoadHostState(%s) attempt/error = %v/%q, want preserved attempt/%q", fixture.serial, state.LastAttemptAt, state.CollectionError, fixture.collectionError)
		}
	}
}

func TestHostDeleteCascadesMunkiReportState(t *testing.T) {
	db, ctx := testdb.Open(t)
	host := munkiReportTestHost(t, ctx, db, "C02MUNKICASCADE")
	store := munki.NewStore(db)
	if err := store.ApplyEnvelope(ctx, munki.EnvelopeResult{
		HostID:      host.ID,
		AttemptedAt: time.Date(2026, 8, 2, 1, 0, 0, 0, time.UTC),
		Complete:    true,
	}); err != nil {
		t.Fatalf("apply envelope: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `DELETE FROM hosts WHERE id = $1`, host.ID); err != nil {
		t.Fatalf("delete host: %v", err)
	}
	var count int
	if err := db.Pool().QueryRow(ctx, `SELECT count(*) FROM munki_host_status WHERE host_id = $1`, host.ID).Scan(&count); err != nil {
		t.Fatalf("count Munki host status: %v", err)
	}
	if count != 0 {
		t.Fatalf("Munki host status rows = %d, want 0", count)
	}
}

func TestApplyEnvelopeReplacesReportAndItems(t *testing.T) {
	db, ctx := testdb.Open(t)
	host := munkiReportTestHost(t, ctx, db, "C02MUNKIENVELOPEREPLACE")
	store := munki.NewStore(db)
	seedMunkiEnvelope(t, ctx, store, host.ID)
	attemptedAt := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	err := store.ApplyEnvelope(ctx, munki.EnvelopeResult{
		HostID: host.ID, AttemptedAt: attemptedAt, Complete: true, HasReport: true,
		Observation: munki.HostObservation{HostID: host.ID, Version: "7.2", ManifestName: "new-manifest"},
		Items:       []munki.ItemObservation{{HostID: host.ID, Name: "Firefox"}},
	})
	if err != nil {
		t.Fatalf("ApplyEnvelope: %v", err)
	}
	state := loadMunkiReportState(t, ctx, store, host.ID)
	if state.Version != "7.2" || state.ManifestName != "new-manifest" || !state.HasReport || state.CollectionError != "" ||
		state.LastAttemptAt == nil || !state.LastAttemptAt.Equal(attemptedAt) || state.LastSuccessfulAt == nil || !state.LastSuccessfulAt.Equal(attemptedAt) {
		t.Fatalf("state = %+v, want replaced successful envelope", state)
	}
	if names := munkiEnvelopeItemNames(t, ctx, db, host.ID); len(names) != 1 || names[0] != "Firefox" {
		t.Fatalf("item names = %#v, want Firefox only", names)
	}
}

func TestApplyReportNoReportClearsItems(t *testing.T) {
	db, ctx := testdb.Open(t)
	host := munkiReportTestHost(t, ctx, db, "C02MUNKIENVELOPENOREPORT")
	store := munki.NewStore(db)
	seedMunkiEnvelope(t, ctx, store, host.ID)
	attemptedAt := time.Date(2026, 8, 2, 11, 0, 0, 0, time.UTC)

	if err := store.ApplyEnvelope(ctx, munki.EnvelopeResult{HostID: host.ID, AttemptedAt: attemptedAt, Complete: true}); err != nil {
		t.Fatalf("ApplyEnvelope: %v", err)
	}
	state := loadMunkiReportState(t, ctx, store, host.ID)
	if state.HasReport || state.Version != "" || state.ManifestName != "" || state.LastSuccessfulAt == nil || !state.LastSuccessfulAt.Equal(attemptedAt) {
		t.Fatalf("state = %+v, want successful no-report envelope", state)
	}
	if names := munkiEnvelopeItemNames(t, ctx, db, host.ID); len(names) != 0 {
		t.Fatalf("item names = %#v, want empty", names)
	}
}

func TestApplyEnvelopeFailedAttemptRetainsSuccessfulFacts(t *testing.T) {
	db, ctx := testdb.Open(t)
	host := munkiReportTestHost(t, ctx, db, "C02MUNKIENVELOPEFAILED")
	store := munki.NewStore(db)
	seedMunkiEnvelope(t, ctx, store, host.ID)
	before := loadMunkiReportState(t, ctx, store, host.ID)
	attemptedAt := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

	if err := store.ApplyEnvelope(ctx, munki.EnvelopeResult{
		HostID: host.ID, AttemptedAt: attemptedAt, CollectionError: "munki_installs: missing result",
	}); err != nil {
		t.Fatalf("ApplyEnvelope: %v", err)
	}
	state := loadMunkiReportState(t, ctx, store, host.ID)
	if state.LastAttemptAt == nil || !state.LastAttemptAt.Equal(attemptedAt) || state.LastSuccessfulAt == nil || before.LastSuccessfulAt == nil || !state.LastSuccessfulAt.Equal(*before.LastSuccessfulAt) ||
		state.CollectionError != "munki_installs: missing result" || state.Version != before.Version || state.ManifestName != before.ManifestName || !state.HasReport {
		t.Fatalf("state = %+v, want retained successful facts with failed attempt", state)
	}
	if names := munkiEnvelopeItemNames(t, ctx, db, host.ID); len(names) != 2 {
		t.Fatalf("item names = %#v, want retained items", names)
	}
}

func TestApplyEnvelopeLaterSuccessClearsError(t *testing.T) {
	db, ctx := testdb.Open(t)
	host := munkiReportTestHost(t, ctx, db, "C02MUNKIENVELOPELATER")
	store := munki.NewStore(db)
	seedMunkiEnvelope(t, ctx, store, host.ID)
	failedAt := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	if err := store.ApplyEnvelope(ctx, munki.EnvelopeResult{HostID: host.ID, AttemptedAt: failedAt, CollectionError: "munki_info: unavailable"}); err != nil {
		t.Fatalf("failed ApplyEnvelope: %v", err)
	}
	successAt := failedAt.Add(time.Hour)
	if err := store.ApplyEnvelope(ctx, munki.EnvelopeResult{
		HostID: host.ID, AttemptedAt: successAt, Complete: true, HasReport: true,
		Observation: munki.HostObservation{HostID: host.ID, Version: "7.3"},
	}); err != nil {
		t.Fatalf("successful ApplyEnvelope: %v", err)
	}
	state := loadMunkiReportState(t, ctx, store, host.ID)
	if state.CollectionError != "" || state.LastAttemptAt == nil || !state.LastAttemptAt.Equal(successAt) || state.LastSuccessfulAt == nil || !state.LastSuccessfulAt.Equal(successAt) || state.Version != "7.3" {
		t.Fatalf("state = %+v, want later successful envelope", state)
	}
}

func TestApplyEnvelopeRollsBackReportAndItemsTogether(t *testing.T) {
	db, ctx := testdb.Open(t)
	host := munkiReportTestHost(t, ctx, db, "C02MUNKIENVELOPEROLLBACK")
	store := munki.NewStore(db)
	seedMunkiEnvelope(t, ctx, store, host.ID)

	err := store.ApplyEnvelope(ctx, munki.EnvelopeResult{
		HostID: host.ID, AttemptedAt: time.Date(2026, 8, 2, 13, 0, 0, 0, time.UTC), Complete: true, HasReport: true,
		Observation: munki.HostObservation{HostID: host.ID, Version: "must-not-persist"},
		Items: []munki.ItemObservation{
			{HostID: host.ID, Name: "duplicate"},
			{HostID: host.ID, Name: "duplicate"},
		},
	})
	if err == nil {
		t.Fatal("ApplyEnvelope succeeded with duplicate item names")
	}
	state := loadMunkiReportState(t, ctx, store, host.ID)
	if state.Version != "seed-version" {
		t.Fatalf("state = %+v, want original report after rollback", state)
	}
	if names := munkiEnvelopeItemNames(t, ctx, db, host.ID); len(names) != 2 {
		t.Fatalf("item names = %#v, want original items after rollback", names)
	}
}

func seedMunkiEnvelope(t *testing.T, ctx context.Context, store *munki.Store, hostID int64) {
	t.Helper()
	seedAt := time.Date(2026, 8, 2, 9, 0, 0, 0, time.UTC)
	if err := store.ApplyEnvelope(ctx, munki.EnvelopeResult{
		HostID: hostID, AttemptedAt: seedAt, Complete: true, HasReport: true,
		Observation: munki.HostObservation{HostID: hostID, Version: "seed-version", ManifestName: "seed-manifest"},
		Items:       []munki.ItemObservation{{HostID: hostID, Name: "old-one"}, {HostID: hostID, Name: "old-two"}},
	}); err != nil {
		t.Fatalf("seed ApplyEnvelope: %v", err)
	}
}

func loadMunkiReportState(t *testing.T, ctx context.Context, store *munki.Store, hostID int64) *munki.HostState {
	t.Helper()
	state, err := store.LoadHostState(ctx, hostID)
	if err != nil {
		t.Fatalf("LoadHostState: %v", err)
	}
	if state == nil {
		t.Fatal("LoadHostState = nil")
	}
	return state
}

func munkiEnvelopeItemNames(t *testing.T, ctx context.Context, db *database.DB, hostID int64) []string {
	t.Helper()
	rows, err := db.Pool().Query(ctx, `SELECT name FROM munki_host_items WHERE host_id = $1 ORDER BY name`, hostID)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan item: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate items: %v", err)
	}
	return names
}

func munkiReportTestHost(t *testing.T, ctx context.Context, db *database.DB, serial string) *hosts.Host {
	t.Helper()
	host, err := hosts.NewStore(db).UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: serial + "-uuid", Serial: serial},
		OrbitNodeKey: serial + "-node-key",
	})
	if err != nil {
		t.Fatalf("enroll host %s: %v", serial, err)
	}
	return host
}
