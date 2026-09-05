//go:build postgres

package hosts

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestApplyInventoryAcceptsBigMemory(t *testing.T) {
	store, ctx := newPostgresHostStore(t)

	host, err := store.UpsertOnOsqueryEnroll(ctx, InventoryUpdate{
		Hardware:       HostHardware{UUID: "test-apply-detail-big-memory"},
		OsqueryNodeKey: "node-key",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	const memoryBytes = int64(68719476736)
	if err := store.ApplyInventory(ctx, host.ID, InventoryUpdate{
		Hardware: HostHardware{MemoryBytes: memoryBytes},
	}); err != nil {
		t.Fatalf("apply inventory: %v", err)
	}

	got, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	if got.Hardware.MemoryBytes != memoryBytes {
		t.Fatalf("memory_bytes = %d, want %d", got.Hardware.MemoryBytes, memoryBytes)
	}
}

func TestInventoryRefreshRequestLifecycle(t *testing.T) {
	store, ctx := newPostgresHostStore(t)

	host, err := store.UpsertOnOsqueryEnroll(ctx, InventoryUpdate{
		Hardware:       HostHardware{UUID: "test-inventory-refresh-request"},
		OsqueryNodeKey: "test-inventory-refresh-request-node-key",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := store.MarkInventoryFresh(ctx, host.ID, "test-query-hash"); err != nil {
		t.Fatalf("mark inventory fresh: %v", err)
	}
	before, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get fresh host: %v", err)
	}
	if before.InventoryUpdatedAt == nil || before.InventoryRefreshRequested {
		t.Fatalf(
			"fresh inventory state = updated %v, requested %t; want timestamp and no request",
			before.InventoryUpdatedAt,
			before.InventoryRefreshRequested,
		)
	}

	if err := store.RequestInventoryRefresh(ctx, host.ID); err != nil {
		t.Fatalf("request inventory refresh: %v", err)
	}
	if err := store.RequestInventoryRefresh(ctx, host.ID); err != nil {
		t.Fatalf("repeat inventory refresh request: %v", err)
	}
	requested, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host with refresh request: %v", err)
	}
	if !requested.InventoryRefreshRequested {
		t.Fatal("InventoryRefreshRequested = false, want true")
	}
	if requested.InventoryUpdatedAt == nil || !requested.InventoryUpdatedAt.Equal(*before.InventoryUpdatedAt) {
		t.Fatalf(
			"InventoryUpdatedAt = %v, want unchanged %v",
			requested.InventoryUpdatedAt,
			before.InventoryUpdatedAt,
		)
	}

	if err := store.MarkInventoryFresh(ctx, host.ID, "test-query-hash"); err != nil {
		t.Fatalf("complete requested inventory refresh: %v", err)
	}
	completed, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get refreshed host: %v", err)
	}
	if completed.InventoryRefreshRequested {
		t.Fatal("InventoryRefreshRequested = true after refresh, want false")
	}

	if err := store.RequestInventoryRefresh(ctx, 999999); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("request missing host error = %v, want ErrNotFound", err)
	}
}

func TestLoadDetailResolvesPrimaryUserFromSourceEmail(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	primaryUsers := NewPrimaryUserStore(store.pool, labels.NewStore(store.pool))

	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-primary-user-direct-user"},
		OrbitNodeKey: "test-primary-user-direct-user-orbit",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if _, err := store.pool.Exec(ctx, `
INSERT INTO users (email, name, password_hash)
VALUES ('test1@woodleigh.vic.edu.au', 'Local Test One', 'password-hash')`); err != nil {
		t.Fatalf("insert same-email local user: %v", err)
	}
	if _, err := store.pool.Exec(ctx, `
INSERT INTO users (
	email, name, source, external_id, user_principal_name,
	mail_nickname, given_name, family_name, department
)
VALUES (
	'test1@woodleigh.vic.edu.au',
	'Test One',
	'entra',
	'test1-entra',
	'test1@woodleigh.vic.edu.au',
	'test1',
	'Test',
	'One',
	'Students'
)`); err != nil {
		t.Fatalf("insert directory user: %v", err)
	}
	if err := primaryUsers.Upsert(
		ctx,
		host.ID,
		"test1@woodleigh.vic.edu.au",
		PrimaryUserSourceOrbitProfile,
	); err != nil {
		t.Fatalf("seed primary user: %v", err)
	}

	detail, err := store.LoadDetail(ctx, host)
	if err != nil {
		t.Fatalf("load detail: %v", err)
	}
	primaryUser := detail.PrimaryUser
	if primaryUser == nil {
		t.Fatal("PrimaryUser is nil")
	}
	if primaryUser.Email != "test1@woodleigh.vic.edu.au" ||
		primaryUser.Username != "test1" ||
		primaryUser.Name != "Test One" ||
		primaryUser.Department != "Students" ||
		primaryUser.Source != PrimaryUserSourceOrbitProfile {
		t.Fatalf("PrimaryUser = %+v, want enriched test1 orbit primary user", primaryUser)
	}
}

func TestPrimaryUserManualSourceOverridesReportedSource(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	primaryUsers := NewPrimaryUserStore(store.pool, labels.NewStore(store.pool))

	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-primary-user-manual-override"},
		OrbitNodeKey: "test-primary-user-manual-override-orbit",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if _, err := store.pool.Exec(ctx, `
INSERT INTO users (
	email, name, source, external_id, user_principal_name,
	mail_nickname, department
)
VALUES
	('reported-one@example.test', 'Reported One', 'entra', 'reported-one', 'reported-one@example.test', 'reported-one', 'Students'),
	('reported-two@example.test', 'Reported Two', 'entra', 'reported-two', 'reported-two@example.test', 'reported-two', 'Staff'),
	('manual@example.test', 'Manual User', 'entra', 'manual-user', 'manual@example.test', 'manual', 'Operations')`); err != nil {
		t.Fatalf("insert directory users: %v", err)
	}

	if err := primaryUsers.Upsert(
		ctx,
		host.ID,
		"reported-one@example.test",
		PrimaryUserSourceOrbitProfile,
	); err != nil {
		t.Fatalf("seed reported primary user: %v", err)
	}
	expectPrimaryUser(
		t,
		ctx,
		store,
		host.ID,
		"reported-one@example.test",
		PrimaryUserSourceOrbitProfile,
		"reported-one",
	)

	if err := primaryUsers.Upsert(
		ctx,
		host.ID,
		"manual@example.test",
		PrimaryUserSourceManual,
	); err != nil {
		t.Fatalf("set manual primary user: %v", err)
	}
	expectPrimaryUser(
		t,
		ctx,
		store,
		host.ID,
		"manual@example.test",
		PrimaryUserSourceManual,
		"manual",
	)

	if err := primaryUsers.Upsert(
		ctx,
		host.ID,
		"reported-two@example.test",
		PrimaryUserSourceOrbitProfile,
	); err != nil {
		t.Fatalf("update reported primary user: %v", err)
	}
	expectPrimaryUser(
		t,
		ctx,
		store,
		host.ID,
		"manual@example.test",
		PrimaryUserSourceManual,
		"manual",
	)

	if err := primaryUsers.Delete(ctx, host.ID, PrimaryUserSourceManual); err != nil {
		t.Fatalf("clear manual primary user: %v", err)
	}
	expectPrimaryUser(
		t,
		ctx,
		store,
		host.ID,
		"reported-two@example.test",
		PrimaryUserSourceOrbitProfile,
		"reported-two",
	)
}

func TestPrimaryUserStoreReturnsNotFoundForMissingHost(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	primaryUsers := NewPrimaryUserStore(store.pool, labels.NewStore(store.pool))

	if err := primaryUsers.Upsert(
		ctx,
		999999,
		"missing@example.test",
		PrimaryUserSourceManual,
	); !errors.Is(
		err,
		fault.ErrNotFound,
	) {
		t.Fatalf("Upsert missing host error = %v, want ErrNotFound", err)
	}
	if err := primaryUsers.Delete(ctx, 999999, PrimaryUserSourceManual); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("Delete missing host error = %v, want ErrNotFound", err)
	}
}

func TestPrimaryUserStoreRollsBackWhenDerivedLabelsCannotRefresh(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-primary-user-refresh-rollback"},
		OrbitNodeKey: "test-primary-user-refresh-rollback-orbit",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if _, err := store.pool.Exec(ctx, `
INSERT INTO labels (name, criteria, label_type, label_membership_type)
VALUES ('Invalid derived label', '{"attribute":"invalid","values":["value"]}', 'regular', 'derived')`); err != nil {
		t.Fatalf("insert invalid derived label: %v", err)
	}
	primaryUsers := NewPrimaryUserStore(store.pool, labels.NewStore(store.pool))

	err = primaryUsers.Upsert(ctx, host.ID, "rollback@example.test", PrimaryUserSourceManual)
	if err == nil {
		t.Fatal("upsert succeeded despite derived label refresh failure")
	}

	var count int
	if err := store.pool.QueryRow(ctx, `
SELECT count(*)
FROM host_primary_user_sources
WHERE host_id = $1 AND source = 'manual'`, host.ID).Scan(&count); err != nil {
		t.Fatalf("count rolled-back primary users: %v", err)
	}
	if count != 0 {
		t.Fatalf("persisted primary users = %d, want 0", count)
	}
}

func TestPrimaryUserStoreRefreshesOnlyTheAffectedHost(t *testing.T) {
	database, ctx := testdb.Open(t)
	labelStore := labels.NewStore(database)
	store := NewStore(database, labelStore)
	primaryUsers := NewPrimaryUserStore(database, labelStore)

	createHost := func(hardwareUUID string) *Host {
		t.Helper()
		host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
			Hardware:     HostHardware{UUID: hardwareUUID},
			OrbitNodeKey: hardwareUUID + "-orbit",
		}, heartbeats.Contact{})

		if err != nil {
			t.Fatalf("enroll host %q: %v", hardwareUUID, err)
		}
		return host
	}
	affected := createHost("primary-user-refresh-affected")
	unrelated := createHost("primary-user-refresh-unrelated")
	if _, err := database.Exec(ctx, `
INSERT INTO users (email, name, source, external_id, user_principal_name, department)
VALUES
	('engineering@example.test', 'Engineering User', 'entra', 'engineering-user', 'engineering@example.test', 'Engineering'),
	('operations@example.test', 'Operations User', 'entra', 'operations-user', 'operations@example.test', 'Operations')`); err != nil {
		t.Fatalf("insert directory users: %v", err)
	}
	for _, hostID := range []int64{affected.ID, unrelated.ID} {
		if err := primaryUsers.Upsert(
			ctx,
			hostID,
			"engineering@example.test",
			PrimaryUserSourceManual,
		); err != nil {
			t.Fatalf("seed host %d primary user: %v", hostID, err)
		}
	}
	derivedLabel, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Engineering primary users",
		LabelMembershipType: labels.LabelMembershipTypeDerived,
		Criteria: &labels.Criteria{
			Attribute: labels.DerivedAttributeUserDepartment,
			Values:    []string{"Engineering"},
		},
	})
	if err != nil {
		t.Fatalf("create derived label: %v", err)
	}
	if _, err := database.Exec(ctx, `
UPDATE host_primary_user_sources
SET email = 'operations@example.test'
WHERE host_id = $1 AND source = 'manual'`, unrelated.ID); err != nil {
		t.Fatalf("make unrelated membership stale: %v", err)
	}
	if err := primaryUsers.Upsert(
		ctx,
		affected.ID,
		"operations@example.test",
		PrimaryUserSourceManual,
	); err != nil {
		t.Fatalf("change affected primary user: %v", err)
	}

	assertMembership := func(hostID int64, want bool) {
		t.Helper()
		var got bool
		if err := database.QueryRow(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM label_membership
	WHERE label_id = $1 AND host_id = $2
)`, derivedLabel.ID, hostID).Scan(&got); err != nil {
			t.Fatalf("lookup host %d membership: %v", hostID, err)
		}
		if got != want {
			t.Fatalf("host %d membership = %t, want %t", hostID, got, want)
		}
	}
	assertMembership(affected.ID, false)
	assertMembership(unrelated.ID, true)
}

// New hosts land in All Hosts.
func TestEnrollAddsHostToAllHosts(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	labelStore := labels.NewStore(store.pool)

	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-enroll-all-hosts"},
		OrbitNodeKey: "orbit-key",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	hostLabels, err := labelStore.ListForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("list labels for host: %v", err)
	}

	var found bool
	for _, l := range hostLabels {
		if l.BuiltinKey != nil &&
			*l.BuiltinKey == labels.BuiltinKeyAllHosts &&
			l.LabelType == labels.LabelTypeBuiltin &&
			l.LabelMembershipType == labels.LabelMembershipTypeManual {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("All Hosts membership missing; got labels = %+v", hostLabels)
	}
}

func TestReenrollPreservesHostIdentity(t *testing.T) {
	store, ctx := newPostgresHostStore(t)

	const hardwareUUID = "test-reenroll-preserves-host-identity"
	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: hardwareUUID},
		OrbitNodeKey: "orbit-key-initial",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host with Orbit: %v", err)
	}
	attached, err := store.UpsertOnOsqueryEnroll(ctx, InventoryUpdate{
		Hardware:       HostHardware{UUID: hardwareUUID},
		OsqueryNodeKey: "osquery-key-initial",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host with osquery: %v", err)
	}
	if attached.ID != host.ID {
		t.Fatalf("first osquery enrollment host ID = %d, want Orbit host ID %d", attached.ID, host.ID)
	}

	orbitReenrolled, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: hardwareUUID},
		OrbitNodeKey: "orbit-key-reenrolled",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("re-enroll host with Orbit: %v", err)
	}
	if orbitReenrolled.ID != host.ID {
		t.Fatalf("Orbit re-enrollment host ID = %d, want existing ID %d", orbitReenrolled.ID, host.ID)
	}
	if _, err := store.GetByOrbitNodeKey(ctx, "orbit-key-initial"); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("get initial Orbit node key error = %v, want ErrNotFound", err)
	}
	if got, err := store.GetByOrbitNodeKey(ctx, "orbit-key-reenrolled"); err != nil || got.ID != host.ID {
		t.Fatalf("get re-enrolled Orbit node key = host %+v, error %v, want host %d", got, err, host.ID)
	}
	if got, err := store.GetByOsqueryNodeKey(ctx, "osquery-key-initial"); err != nil || got.ID != host.ID {
		t.Fatalf("get preserved osquery node key = host %+v, error %v, want host %d", got, err, host.ID)
	}

	osqueryReenrolled, err := store.UpsertOnOsqueryEnroll(ctx, InventoryUpdate{
		Hardware:       HostHardware{UUID: hardwareUUID},
		OsqueryNodeKey: "osquery-key-reenrolled",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("re-enroll host with osquery: %v", err)
	}
	if osqueryReenrolled.ID != host.ID {
		t.Fatalf("osquery re-enrollment host ID = %d, want existing ID %d", osqueryReenrolled.ID, host.ID)
	}
	if _, err := store.GetByOsqueryNodeKey(ctx, "osquery-key-initial"); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("get initial osquery node key error = %v, want ErrNotFound", err)
	}
	if got, err := store.GetByOsqueryNodeKey(ctx, "osquery-key-reenrolled"); err != nil || got.ID != host.ID {
		t.Fatalf("get re-enrolled osquery node key = host %+v, error %v, want host %d", got, err, host.ID)
	}
	if got, err := store.GetByOrbitNodeKey(ctx, "orbit-key-reenrolled"); err != nil || got.ID != host.ID {
		t.Fatalf("get preserved Orbit node key = host %+v, error %v, want host %d", got, err, host.ID)
	}
}

func TestReenrollRollsBackHostMutationWhenContactIsInvalid(t *testing.T) {
	store, ctx := newPostgresHostStore(t)

	const hardwareUUID = "test-reenroll-contact-rollback"
	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: hardwareUUID},
		OrbitNodeKey: "orbit-key-before-invalid-contact",
	}, heartbeats.Contact{RemoteIP: "192.0.2.1", UserAgent: "Orbit/1.0"})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	_, err = store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: hardwareUUID},
		OrbitNodeKey: "orbit-key-after-invalid-contact",
	}, heartbeats.Contact{RemoteIP: "not-an-ip", UserAgent: "Orbit/2.0"})
	if !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("re-enroll error = %v, want ErrInvalidInput", err)
	}
	if got, err := store.GetByOrbitNodeKey(ctx, "orbit-key-before-invalid-contact"); err != nil || got.ID != host.ID {
		t.Fatalf("get original Orbit node key = host %+v, error %v, want host %d", got, err, host.ID)
	}
	if _, err := store.GetByOrbitNodeKey(ctx, "orbit-key-after-invalid-contact"); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("get rolled-back Orbit node key error = %v, want ErrNotFound", err)
	}

	got, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host after rollback: %v", err)
	}
	if len(got.Heartbeats) != 1 || got.Heartbeats[0].UserAgent != "Orbit/1.0" {
		t.Fatalf("heartbeats after rollback = %+v, want original Orbit contact", got.Heartbeats)
	}
}

func TestGetByHardwareSerialRequiresUniqueRealSerial(t *testing.T) {
	store, ctx := newPostgresHostStore(t)

	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-serial-identity-1", Serial: "C02SERIAL"},
		OrbitNodeKey: "orbit-key-serial-1",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	got, err := store.GetByHardwareSerial(ctx, " C02SERIAL ")
	if err != nil {
		t.Fatalf("get by hardware serial: %v", err)
	}
	if got.ID != host.ID {
		t.Fatalf("host id = %d, want %d", got.ID, host.ID)
	}

	if _, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-serial-identity-2", Serial: "C02SERIAL"},
		OrbitNodeKey: "orbit-key-serial-2",
	}, heartbeats.Contact{}); err == nil {
		t.Fatal("duplicate hardware serial insert succeeded")
	}
}

func TestHostProjectionUsesSourceSpecificHeartbeats(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	host, err := store.UpsertOnOsqueryEnroll(ctx, InventoryUpdate{
		Hardware:        HostHardware{UUID: "test-heartbeat-projection"},
		OsqueryNodeKey:  "test-heartbeat-projection-osquery",
		OrbitNodeKey:    "test-heartbeat-projection-orbit",
		LastRestartedAt: new(now.Add(-time.Hour)),
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	recordTestHeartbeat(t, ctx, store, host.ID, heartbeats.SourceOsquery, now.Add(-4*time.Minute), "198.51.100.10", "osquery/5.14")
	recordTestHeartbeat(t, ctx, store, host.ID, heartbeats.SourceSanta, now.Add(-time.Minute), "198.51.100.20", "santa/2026.4")
	recordTestHeartbeat(t, ctx, store, host.ID, heartbeats.SourceMunki, now.Add(-2*time.Minute), "198.51.100.30", "managedsoftwareupdate/6.6")

	got, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	if got.Status != HostStatusOnline {
		t.Fatalf("status = %q, want online from osquery heartbeat", got.Status)
	}
	if got.PublicIP == nil || *got.PublicIP != netip.MustParseAddr("198.51.100.10") {
		t.Fatalf("public_ip = %v, want osquery IP", got.PublicIP)
	}
	if got.LastContact == nil || !got.LastContact.Equal(now.Add(-time.Minute)) {
		t.Fatalf("last_contact = %v, want newest heartbeat", got.LastContact)
	}
	wantSources := []heartbeats.Source{heartbeats.SourceMunki, heartbeats.SourceOsquery, heartbeats.SourceSanta}
	if len(got.Heartbeats) != len(wantSources) {
		t.Fatalf("heartbeats = %+v, want %d", got.Heartbeats, len(wantSources))
	}
	for i, source := range wantSources {
		if got.Heartbeats[i].Source != source {
			t.Fatalf("heartbeats[%d].source = %q, want %q", i, got.Heartbeats[i].Source, source)
		}
	}

	enrolled, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-enrollment-heartbeat"},
		OrbitNodeKey: "test-enrollment-heartbeat-orbit",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll heartbeat host: %v", err)
	}
	if len(enrolled.Heartbeats) != 1 || enrolled.Heartbeats[0].Source != heartbeats.SourceOrbit {
		t.Fatalf("heartbeats = %#v, want Orbit enrollment heartbeat", enrolled.Heartbeats)
	}
}

func TestNodeKeyAndTokenLookupsAreIdentityReads(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	if _, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-identity-only-lookups"},
		OrbitNodeKey: "test-identity-only-orbit",
	}, heartbeats.Contact{}); err != nil {
		t.Fatalf("enroll host with Orbit: %v", err)
	}
	host, err := store.UpsertOnOsqueryEnroll(ctx, InventoryUpdate{
		Hardware:       HostHardware{UUID: "test-identity-only-lookups"},
		OsqueryNodeKey: "test-identity-only-osquery",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host with osquery: %v", err)
	}
	const token = "00000000-0000-4000-8000-000000000001"
	if err := store.SetOrbitDeviceAuthToken(ctx, host.OrbitNodeKey, token); err != nil {
		t.Fatalf("set device token: %v", err)
	}
	before, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host before lookups: %v", err)
	}
	lookups := []struct {
		name string
		load func() (*Host, error)
	}{
		{name: "orbit node key", load: func() (*Host, error) { return store.GetByOrbitNodeKey(ctx, host.OrbitNodeKey) }},
		{name: "osquery node key", load: func() (*Host, error) { return store.GetByOsqueryNodeKey(ctx, host.OsqueryNodeKey) }},
		{name: "orbit token", load: func() (*Host, error) { return store.ValidateOrbitDeviceAuthToken(ctx, token) }},
	}
	for _, lookup := range lookups {
		if _, err := lookup.load(); err != nil {
			t.Fatalf("%s: %v", lookup.name, err)
		}
	}
	after, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host after lookups: %v", err)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("identity lookups changed updated_at from %v to %v", before.UpdatedAt, after.UpdatedAt)
	}
	if before.LastContact == nil || after.LastContact == nil || !after.LastContact.Equal(*before.LastContact) {
		t.Fatalf("identity lookups changed last contact from %v to %v", before.LastContact, after.LastContact)
	}
	if len(after.Heartbeats) != len(before.Heartbeats) {
		t.Fatalf("identity lookups changed heartbeats from %+v to %+v", before.Heartbeats, after.Heartbeats)
	}
	for i := range before.Heartbeats {
		if after.Heartbeats[i].Source != before.Heartbeats[i].Source ||
			!after.Heartbeats[i].LastSeenAt.Equal(before.Heartbeats[i].LastSeenAt) {
			t.Fatalf("identity lookups changed heartbeats from %+v to %+v", before.Heartbeats, after.Heartbeats)
		}
	}
}

func TestHostListFiltersAndSortsByFlattenedContactFields(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	online := enrollTestHost(t, ctx, store, "test-list-online", new(now.Add(-2*time.Hour)))
	offline := enrollTestHost(t, ctx, store, "test-list-offline", new(now.Add(-time.Hour)))
	missing := enrollTestHost(t, ctx, store, "test-list-missing-osquery", nil)
	if _, err := store.pool.Exec(ctx, `DELETE FROM host_heartbeats WHERE host_id = ANY($1)`, []int64{
		online.ID,
		offline.ID,
		missing.ID,
	}); err != nil {
		t.Fatalf("clear enrollment heartbeats: %v", err)
	}
	recordTestHeartbeat(t, ctx, store, online.ID, heartbeats.SourceOsquery, now.Add(-time.Minute), "198.51.100.20", "")
	recordTestHeartbeat(t, ctx, store, offline.ID, heartbeats.SourceOsquery, now.Add(-10*time.Minute), "198.51.100.10", "")
	recordTestHeartbeat(t, ctx, store, offline.ID, heartbeats.SourceSanta, now, "203.0.113.50", "")
	recordTestHeartbeat(t, ctx, store, missing.ID, heartbeats.SourceMunki, now.Add(-30*time.Second), "203.0.113.60", "")

	onlineRows, _, err := store.List(ctx, HostListParams{Status: HostStatusOnline})
	if err != nil {
		t.Fatalf("list online hosts: %v", err)
	}
	if !containsHostID(onlineRows, online.ID) || containsHostID(onlineRows, offline.ID) || containsHostID(onlineRows, missing.ID) {
		t.Fatalf("online list = %+v, want only recent osquery host", hostIDs(onlineRows))
	}
	offlineRows, _, err := store.List(ctx, HostListParams{Status: HostStatusOffline})
	if err != nil {
		t.Fatalf("list offline hosts: %v", err)
	}
	if containsHostID(offlineRows, online.ID) || !containsHostID(offlineRows, offline.ID) || !containsHostID(offlineRows, missing.ID) {
		t.Fatalf("offline list = %+v, want stale/missing osquery hosts", hostIDs(offlineRows))
	}
	for _, host := range offlineRows {
		if host.ID == offline.ID && host.Status != HostStatusOffline {
			t.Fatalf("stale osquery host status = %q, want offline despite newer Santa heartbeat", host.Status)
		}
	}

	tests := []struct {
		sort string
		want int64
	}{
		{sort: "last_contact.desc", want: offline.ID},
		{sort: "last_restarted_at.desc", want: offline.ID},
		{sort: "public_ip.asc", want: offline.ID},
	}
	for _, test := range tests {
		rows, _, err := store.List(ctx, HostListParams{ListParams: listing.Params{Sort: test.sort}})
		if err != nil {
			t.Fatalf("sort %q: %v", test.sort, err)
		}
		if len(rows) < 2 || rows[0].ID != test.want {
			t.Fatalf("sort %q ids = %v, first = %d", test.sort, hostIDs(rows), test.want)
		}
	}
}

func TestHostListSearchesPersistedIdentityNetworkAndPrimaryUserFields(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	primaryUsers := NewPrimaryUserStore(store.pool, labels.NewStore(store.pool))
	host, err := store.UpsertOnOsqueryEnroll(ctx, InventoryUpdate{
		ComputerName:   "Searchable Mac",
		Hardware:       HostHardware{UUID: "test-search-host", Serial: "C02SEARCH123"},
		OsqueryNodeKey: "test-search-node-key",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := store.ApplyInventory(ctx, host.ID, InventoryUpdate{
		Hostname:     "searchable-mac.woodleigh.local",
		ComputerName: "Searchable Mac",
		Hardware: HostHardware{
			Serial:          "C02SEARCH123",
			Vendor:          "Apple Search Vendor",
			ModelIdentifier: "MacSearch10,1",
			CPU: HostCPU{
				Architecture: "arm64-search",
				Subtype:      "search-subtype",
				Brand:        "Search Silicon",
			},
		},
		OS: HostOS{
			Platform:      "darwin-search",
			Name:          "SearchOS",
			Version:       "26.4-search",
			Build:         "25ESEARCH",
			KernelVersion: "search-kernel",
		},
		Network: InventoryNetwork{
			PrimaryIP:  "10.44.55.66",
			PrimaryMAC: "00:11:22:33:44:55",
		},
		Agents: HostAgents{
			Osquery: HostOsqueryAgent{Version: "5.99-search"},
			Orbit:   HostOrbitAgent{Version: "1.99-search"},
		},
	}); err != nil {
		t.Fatalf("apply inventory: %v", err)
	}
	recordTestHeartbeat(
		t, ctx, store, host.ID, heartbeats.SourceOsquery, time.Now(), "198.51.100.77", "",
	)
	if _, err := store.pool.Exec(ctx, `
INSERT INTO users (email, name, password_hash)
VALUES (
	'search.person@woodleigh.vic.edu.au',
	'Local Search Person',
	'password-hash'
)`); err != nil {
		t.Fatalf("insert same-email local user: %v", err)
	}
	if _, err := store.pool.Exec(ctx, `
INSERT INTO users (
	email, name, source, external_id, user_principal_name,
	mail_nickname, given_name, family_name, department
)
VALUES (
	'search.person@woodleigh.vic.edu.au',
	'Search Person',
	'entra',
	'test-search-person-entra',
	'search.person@woodleigh.vic.edu.au',
	'searchperson',
	'Search',
	'Person',
	'Managed Devices'
)`); err != nil {
		t.Fatalf("insert directory user: %v", err)
	}
	if err := primaryUsers.Upsert(
		ctx,
		host.ID,
		"search.person@woodleigh.vic.edu.au",
		PrimaryUserSourceManual,
	); err != nil {
		t.Fatalf("set primary user: %v", err)
	}

	queries := []string{
		"searchable-mac.woodleigh.local",
		"C02SEARCH123",
		"MacSearch10,1",
		"Search Silicon",
		"25ESEARCH",
		"10.44.55.66",
		"198.51.100.77",
		"search.person@woodleigh.vic.edu.au",
		"searchperson",
		"Search Person",
		"Managed Devices",
	}
	for _, query := range queries {
		rows, count, err := store.List(ctx, HostListParams{
			ListParams: listing.Params{Q: query},
		})
		if err != nil {
			t.Fatalf("search %q: %v", query, err)
		}
		if count != 1 || len(rows) != 1 || rows[0].ID != host.ID {
			t.Fatalf("search %q returned ids %v and count %d", query, hostIDs(rows), count)
		}
	}
}

func TestResolveSelectedTargetsMergesDirectHostsAndLabels(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	labelStore := labels.NewStore(store.pool)

	directHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-target-direct"},
		OrbitNodeKey: "orbit-key-direct",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll direct host: %v", err)
	}
	labelHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-target-label"},
		OrbitNodeKey: "orbit-key-label",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll label host: %v", err)
	}
	label, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Live Target Test",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	if err := labelStore.SetMembership(ctx, label.ID, labelHost.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}

	got, err := store.ResolveSelectedTargets(ctx, TargetSelection{
		HostIDs:  []int64{directHost.ID, directHost.ID, -1},
		LabelIDs: []int64{label.ID},
	})
	if err != nil {
		t.Fatalf("resolve selected targets: %v", err)
	}
	if !sameIDs(got, []int64{directHost.ID, labelHost.ID}) {
		t.Fatalf("resolved host ids = %v, want direct and label hosts", got)
	}
}

func TestCountSelectedTargetsSplitsOnlineAndOffline(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	labelStore := labels.NewStore(store.pool)
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)

	onlineHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-count-online"},
		OrbitNodeKey: "orbit-key-count-online",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll online host: %v", err)
	}
	offlineHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-count-offline"},
		OrbitNodeKey: "orbit-key-count-offline",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll offline host: %v", err)
	}
	recordTestHeartbeat(t, ctx, store, onlineHost.ID, heartbeats.SourceOsquery, now.Add(-time.Minute), "", "")
	recordTestHeartbeat(t, ctx, store, offlineHost.ID, heartbeats.SourceOsquery, now.Add(-10*time.Minute), "", "")
	recordTestHeartbeat(t, ctx, store, offlineHost.ID, heartbeats.SourceSanta, now, "", "")
	label, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Live Count Test",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	if err := labelStore.SetMembership(ctx, label.ID, offlineHost.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}

	got, err := store.CountSelectedTargets(ctx, TargetSelection{
		HostIDs:  []int64{onlineHost.ID, offlineHost.ID, onlineHost.ID, -1},
		LabelIDs: []int64{label.ID},
	}, now)
	if err != nil {
		t.Fatalf("count selected targets: %v", err)
	}
	want := TargetMetrics{Total: 2, Online: 1, Offline: 1}
	if got != want {
		t.Fatalf("target metrics = %+v, want %+v", got, want)
	}
}

func TestResolveOnlineSelectedTargetsReturnsOnlyCurrentlyOnlineHosts(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	labelStore := labels.NewStore(store.pool)
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)

	onlineHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-online-target-online"},
		OrbitNodeKey: "orbit-key-live-online",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll online host: %v", err)
	}
	offlineHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-online-target-offline"},
		OrbitNodeKey: "orbit-key-live-offline",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll offline host: %v", err)
	}
	label, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Live Online Target Test",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	if err := labelStore.SetMembership(ctx, label.ID, offlineHost.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}
	recordTestHeartbeat(t, ctx, store, onlineHost.ID, heartbeats.SourceOsquery, now.Add(-time.Minute), "", "")
	recordTestHeartbeat(t, ctx, store, offlineHost.ID, heartbeats.SourceOsquery, now.Add(-10*time.Minute), "", "")
	recordTestHeartbeat(t, ctx, store, offlineHost.ID, heartbeats.SourceMunki, now, "", "")

	got, err := store.ResolveOnlineSelectedTargets(ctx, TargetSelection{
		HostIDs:  []int64{onlineHost.ID},
		LabelIDs: []int64{label.ID},
	}, now)
	if err != nil {
		t.Fatalf("resolve online selected targets: %v", err)
	}
	if len(got) != 1 || got[0].ID != onlineHost.ID || got[0].DisplayName != onlineHost.DisplayName {
		t.Fatalf("online hosts = %+v, want only online host %+v", got, onlineHost)
	}
}

func expectPrimaryUser(
	t *testing.T,
	ctx context.Context,
	store *Store,
	hostID int64,
	wantEmail string,
	wantSource PrimaryUserSource,
	wantUsername string,
) {
	t.Helper()
	host, err := store.GetByID(ctx, hostID)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	detail, err := store.LoadDetail(ctx, host)
	if err != nil {
		t.Fatalf("load detail: %v", err)
	}
	primaryUser := detail.PrimaryUser
	if primaryUser == nil {
		t.Fatal("PrimaryUser is nil")
	}
	if primaryUser.Email != wantEmail || primaryUser.Source != wantSource || primaryUser.Username != wantUsername {
		t.Fatalf(
			"PrimaryUser = %+v, want email %q source %q username %q",
			primaryUser,
			wantEmail,
			wantSource,
			wantUsername,
		)
	}
}

func sameIDs(got []int64, want []int64) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[int64]int, len(got))
	for _, id := range got {
		seen[id]++
	}
	for _, id := range want {
		if seen[id] == 0 {
			return false
		}
		seen[id]--
	}
	return true
}

func enrollTestHost(
	t *testing.T,
	ctx context.Context,
	store *Store,
	hardwareUUID string,
	lastRestartedAt *time.Time,
) *Host {
	t.Helper()
	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:        HostHardware{UUID: hardwareUUID},
		OrbitNodeKey:    hardwareUUID + "-orbit",
		LastRestartedAt: lastRestartedAt,
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll %s: %v", hardwareUUID, err)
	}
	if lastRestartedAt != nil {
		if err := store.ApplyInventory(ctx, host.ID, InventoryUpdate{LastRestartedAt: lastRestartedAt}); err != nil {
			t.Fatalf("set %s restart time: %v", hardwareUUID, err)
		}
	}
	return host
}

func recordTestHeartbeat(
	t *testing.T,
	ctx context.Context,
	store *Store,
	hostID int64,
	source heartbeats.Source,
	lastSeenAt time.Time,
	remoteIP string,
	userAgent string,
) {
	t.Helper()
	if _, err := store.pool.Exec(ctx, `
INSERT INTO host_heartbeats (host_id, source, last_seen_at, remote_ip, user_agent)
VALUES ($1, $2, $3, NULLIF($4, '')::inet, $5)
ON CONFLICT (host_id, source) DO UPDATE SET
	last_seen_at = EXCLUDED.last_seen_at,
	remote_ip = EXCLUDED.remote_ip,
	user_agent = EXCLUDED.user_agent`, hostID, source, lastSeenAt, remoteIP, userAgent); err != nil {
		t.Fatalf("record %s heartbeat: %v", source, err)
	}
}

func containsHostID(hosts []Host, id int64) bool {
	for _, host := range hosts {
		if host.ID == id {
			return true
		}
	}
	return false
}

func hostIDs(hosts []Host) []int64 {
	ids := make([]int64, len(hosts))
	for i, host := range hosts {
		ids[i] = host.ID
	}
	return ids
}

func newPostgresHostStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	database, ctx := testdb.Open(t)
	return NewStore(database, labels.NewStore(database)), ctx
}
