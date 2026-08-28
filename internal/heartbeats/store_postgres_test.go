//go:build postgres

package heartbeats

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestRecordInsertsHeartbeat(t *testing.T) {
	store, db, ctx := newPostgresHeartbeatStore(t)
	hostID := insertHeartbeatHost(t, ctx, db, "first-record")

	if err := store.Record(ctx, hostID, SourceOrbit, Contact{
		RemoteIP:  "192.0.2.1",
		UserAgent: "Orbit/1.0",
	}); err != nil {
		t.Fatalf("record heartbeat: %v", err)
	}

	heartbeat := loadHeartbeat(t, ctx, db, hostID)
	if heartbeat.LastSeenAt.IsZero() {
		t.Fatal("LastSeenAt is zero")
	}
	if heartbeat.RemoteIP == nil || *heartbeat.RemoteIP != netip.MustParseAddr("192.0.2.1") {
		t.Fatalf("RemoteIP = %v, want 192.0.2.1", heartbeat.RemoteIP)
	}
	if heartbeat.UserAgent != "Orbit/1.0" {
		t.Fatalf("UserAgent = %q, want Orbit/1.0", heartbeat.UserAgent)
	}
}

func TestRecordUpdatesCurrentHeartbeat(t *testing.T) {
	store, db, ctx := newPostgresHeartbeatStore(t)
	hostID := insertHeartbeatHost(t, ctx, db, "update-current")

	if err := store.Record(ctx, hostID, SourceOrbit, Contact{
		RemoteIP:  "192.0.2.1",
		UserAgent: "Orbit/1.0",
	}); err != nil {
		t.Fatalf("record first heartbeat: %v", err)
	}
	stale := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	if _, err := db.Exec(ctx, `
		UPDATE host_heartbeats
		SET last_seen_at = $1
		WHERE host_id = $2 AND source = $3`, stale, hostID, SourceOrbit); err != nil {
		t.Fatalf("make heartbeat stale: %v", err)
	}
	if err := store.Record(ctx, hostID, SourceOrbit, Contact{
		RemoteIP:  "2001:db8::1",
		UserAgent: "Orbit/2.0",
	}); err != nil {
		t.Fatalf("record second heartbeat: %v", err)
	}
	second := loadHeartbeat(t, ctx, db, hostID)

	if !second.LastSeenAt.After(stale) {
		t.Fatalf("LastSeenAt = %v, want after %v", second.LastSeenAt, stale)
	}
	if second.RemoteIP == nil || *second.RemoteIP != netip.MustParseAddr("2001:db8::1") {
		t.Fatalf("RemoteIP = %v, want 2001:db8::1", second.RemoteIP)
	}
	if second.UserAgent != "Orbit/2.0" {
		t.Fatalf("UserAgent = %q, want Orbit/2.0", second.UserAgent)
	}
	var count int
	if err := db.QueryRow(ctx, `
		SELECT count(*)::integer
		FROM host_heartbeats
		WHERE host_id = $1 AND source = $2`, hostID, SourceOrbit).Scan(&count); err != nil {
		t.Fatalf("count heartbeats: %v", err)
	}
	if count != 1 {
		t.Fatalf("heartbeat count = %d, want 1", count)
	}
}

func TestRecordSkipsFreshUnchangedHeartbeatWithoutAssigningXID(t *testing.T) {
	store, db, ctx := newPostgresHeartbeatStore(t)
	hostID := insertHeartbeatHost(t, ctx, db, "skip-fresh")
	contact := Contact{RemoteIP: "192.0.2.1", UserAgent: "Orbit/1.0"}
	if err := store.Record(ctx, hostID, SourceOrbit, contact); err != nil {
		t.Fatalf("record first heartbeat: %v", err)
	}
	first := loadHeartbeat(t, ctx, db, hostID)

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	if err := RecordTx(ctx, tx, hostID, SourceOrbit, contact); err != nil {
		t.Fatalf("record fresh heartbeat: %v", err)
	}
	var xidUnassigned bool
	if err := tx.QueryRow(ctx, `SELECT pg_current_xact_id_if_assigned() IS NULL`).Scan(&xidUnassigned); err != nil {
		t.Fatalf("inspect transaction ID: %v", err)
	}
	if !xidUnassigned {
		t.Fatal("fresh unchanged heartbeat assigned a transaction ID")
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}

	second := loadHeartbeat(t, ctx, db, hostID)
	if !second.LastSeenAt.Equal(first.LastSeenAt) {
		t.Fatalf("LastSeenAt = %v, want unchanged %v", second.LastSeenAt, first.LastSeenAt)
	}
}

func TestRecordRefreshesChangedContactImmediately(t *testing.T) {
	store, db, ctx := newPostgresHeartbeatStore(t)
	hostID := insertHeartbeatHost(t, ctx, db, "changed-contact")
	if err := store.Record(ctx, hostID, SourceOrbit, Contact{
		RemoteIP:  "192.0.2.1",
		UserAgent: "Orbit/1.0",
	}); err != nil {
		t.Fatalf("record first heartbeat: %v", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	if err := RecordTx(ctx, tx, hostID, SourceOrbit, Contact{
		RemoteIP:  "2001:db8::1",
		UserAgent: "Orbit/2.0",
	}); err != nil {
		t.Fatalf("record changed contact: %v", err)
	}
	var xidUnassigned bool
	if err := tx.QueryRow(ctx, `SELECT pg_current_xact_id_if_assigned() IS NULL`).Scan(&xidUnassigned); err != nil {
		t.Fatalf("inspect transaction ID: %v", err)
	}
	if xidUnassigned {
		t.Fatal("changed heartbeat contact did not assign a transaction ID")
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}

	heartbeat := loadHeartbeat(t, ctx, db, hostID)
	if heartbeat.RemoteIP == nil || *heartbeat.RemoteIP != netip.MustParseAddr("2001:db8::1") {
		t.Fatalf("RemoteIP = %v, want 2001:db8::1", heartbeat.RemoteIP)
	}
	if heartbeat.UserAgent != "Orbit/2.0" {
		t.Fatalf("UserAgent = %q, want Orbit/2.0", heartbeat.UserAgent)
	}
}

func TestRecordStoresSourcesSeparately(t *testing.T) {
	store, db, ctx := newPostgresHeartbeatStore(t)
	hostID := insertHeartbeatHost(t, ctx, db, "separate-sources")

	if err := store.Record(ctx, hostID, SourceOrbit, Contact{}); err != nil {
		t.Fatalf("record orbit heartbeat: %v", err)
	}
	if err := store.Record(ctx, hostID, SourceOsquery, Contact{}); err != nil {
		t.Fatalf("record osquery heartbeat: %v", err)
	}

	var count int
	if err := db.QueryRow(ctx, `
		SELECT count(*)::integer FROM host_heartbeats WHERE host_id = $1`, hostID).Scan(&count); err != nil {
		t.Fatalf("count heartbeats: %v", err)
	}
	if count != 2 {
		t.Fatalf("heartbeat count = %d, want 2", count)
	}
	if heartbeat := loadHeartbeat(t, ctx, db, hostID); heartbeat.RemoteIP != nil {
		t.Fatalf("RemoteIP = %v, want nil", heartbeat.RemoteIP)
	}
}

func TestRecordRejectsInvalidHostAndSource(t *testing.T) {
	store, db, ctx := newPostgresHeartbeatStore(t)
	hostID := insertHeartbeatHost(t, ctx, db, "validation")

	if err := store.Record(ctx, 0, SourceOrbit, Contact{}); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("Record invalid host error = %v, want ErrInvalidInput", err)
	}
	if err := store.Record(ctx, hostID, Source("other"), Contact{}); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("Record invalid source error = %v, want ErrInvalidInput", err)
	}
}

func TestRecordRejectsInvalidRemoteIP(t *testing.T) {
	store, db, ctx := newPostgresHeartbeatStore(t)
	hostID := insertHeartbeatHost(t, ctx, db, "invalid-remote-ip")

	for _, remoteIP := range []string{"not-an-ip", "192.0.2.1/24", "fe80::1%en0"} {
		err := store.Record(ctx, hostID, SourceOrbit, Contact{RemoteIP: remoteIP})
		if !errors.Is(err, fault.ErrInvalidInput) {
			t.Fatalf("Record remote IP %q error = %v, want ErrInvalidInput", remoteIP, err)
		}
	}
}

func TestHostDeletionCascadesHeartbeats(t *testing.T) {
	store, db, ctx := newPostgresHeartbeatStore(t)
	hostID := insertHeartbeatHost(t, ctx, db, "delete-cascade")

	if err := store.Record(ctx, hostID, SourceSanta, Contact{}); err != nil {
		t.Fatalf("record heartbeat: %v", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM hosts WHERE id = $1`, hostID); err != nil {
		t.Fatalf("delete host: %v", err)
	}
	var count int
	if err := db.QueryRow(ctx, `SELECT count(*)::integer FROM host_heartbeats WHERE host_id = $1`, hostID).Scan(&count); err != nil {
		t.Fatalf("count heartbeats: %v", err)
	}
	if count != 0 {
		t.Fatalf("heartbeat count = %d, want 0", count)
	}
}

func newPostgresHeartbeatStore(t *testing.T) (*Store, *pgxpool.Pool, context.Context) {
	t.Helper()
	db, ctx := testdb.Open(t)
	return NewStore(db), db, ctx
}

func insertHeartbeatHost(t *testing.T, ctx context.Context, db *pgxpool.Pool, hardwareUUID string) int64 {
	t.Helper()
	var hostID int64
	if err := db.QueryRow(ctx, `
		INSERT INTO hosts (hardware_uuid)
		VALUES ($1)
		RETURNING id`, hardwareUUID).Scan(&hostID); err != nil {
		t.Fatalf("insert host: %v", err)
	}
	return hostID
}

func loadHeartbeat(t *testing.T, ctx context.Context, db *pgxpool.Pool, hostID int64) Heartbeat {
	t.Helper()
	var heartbeat Heartbeat
	if err := db.QueryRow(ctx, `
		SELECT source, last_seen_at, remote_ip, user_agent
		FROM host_heartbeats
		WHERE host_id = $1 AND source = $2`, hostID, SourceOrbit).Scan(
		&heartbeat.Source,
		&heartbeat.LastSeenAt,
		&heartbeat.RemoteIP,
		&heartbeat.UserAgent,
	); err != nil {
		t.Fatalf("load heartbeat: %v", err)
	}
	return heartbeat
}
