package hosts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
)

func TestInventoryDisplayNamePriority(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   InventoryUpdate
		want string
	}{
		{
			name: "computer name wins",
			in: InventoryUpdate{
				ComputerName: "Example MacBook Pro",
				Hostname:     "example-macbook-pro",
				Hardware:     HostHardware{UUID: "uuid-1"},
			},
			want: "Example MacBook Pro",
		},
		{
			name: "hostname when no computer name",
			in:   InventoryUpdate{Hostname: "example-macbook-pro", Hardware: HostHardware{UUID: "uuid-1"}},
			want: "example-macbook-pro",
		},
		{
			name: "uuid when no friendly name",
			in:   InventoryUpdate{Hardware: HostHardware{UUID: "uuid-1"}},
			want: "uuid-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := inventoryDisplayName(tt.in.Hardware.UUID, tt.in.Hostname, tt.in.ComputerName); got != tt.want {
				t.Fatalf("inventoryDisplayName = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyInventoryAcceptsBigMemory(t *testing.T) {
	store, ctx := newIntegrationHostStore(t)

	host, err := store.UpsertOnOsqueryEnroll(ctx, InventoryUpdate{
		Hardware:       HostHardware{UUID: "test-apply-detail-big-memory"},
		OsqueryNodeKey: "node-key",
	})
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

func TestLoadDetailResolvesPrimaryUserFromSourceEmail(t *testing.T) {
	store, ctx := newIntegrationHostStore(t)
	primaryUsers := NewPrimaryUserStore(store.db)

	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-primary-user-direct-user"},
		OrbitNodeKey: "test-primary-user-direct-user-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if _, err := store.db.Pool().Exec(ctx, `
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
	store, ctx := newIntegrationHostStore(t)
	primaryUsers := NewPrimaryUserStore(store.db)

	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-primary-user-manual-override"},
		OrbitNodeKey: "test-primary-user-manual-override-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if _, err := store.db.Pool().Exec(ctx, `
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
	store, ctx := newIntegrationHostStore(t)
	primaryUsers := NewPrimaryUserStore(store.db)

	if err := primaryUsers.Upsert(
		ctx,
		999999,
		"missing@example.test",
		PrimaryUserSourceManual,
	); !errors.Is(
		err,
		dbutil.ErrNotFound,
	) {
		t.Fatalf("Upsert missing host error = %v, want ErrNotFound", err)
	}
	if err := primaryUsers.Delete(ctx, 999999, PrimaryUserSourceManual); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("Delete missing host error = %v, want ErrNotFound", err)
	}
}

func TestPrimaryUserStoreRollsBackWhenDerivedLabelsCannotRefresh(t *testing.T) {
	store, ctx := newIntegrationHostStore(t)
	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-primary-user-refresh-rollback"},
		OrbitNodeKey: "test-primary-user-refresh-rollback-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if _, err := store.db.Pool().Exec(ctx, `
INSERT INTO labels (name, criteria, label_type, label_membership_type)
VALUES ('Invalid derived label', '{"attribute":"invalid","values":["value"]}', 'regular', 'derived')`); err != nil {
		t.Fatalf("insert invalid derived label: %v", err)
	}
	primaryUsers := NewPrimaryUserStore(store.db)

	err = primaryUsers.Upsert(ctx, host.ID, "rollback@example.test", PrimaryUserSourceManual)
	if err == nil {
		t.Fatal("upsert succeeded despite derived label refresh failure")
	}

	var count int
	if err := store.db.Pool().QueryRow(ctx, `
SELECT count(*)
FROM host_primary_user_sources
WHERE host_id = $1 AND source = 'manual'`, host.ID).Scan(&count); err != nil {
		t.Fatalf("count rolled-back primary users: %v", err)
	}
	if count != 0 {
		t.Fatalf("persisted primary users = %d, want 0", count)
	}
}

// New hosts land in All Hosts.
func TestEnrollAddsHostToAllHosts(t *testing.T) {
	store, ctx := newIntegrationHostStore(t)
	labelStore := labels.NewStore(store.db)

	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-enroll-all-hosts"},
		OrbitNodeKey: "orbit-key",
	})
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

func TestGetByHardwareSerialRequiresUniqueRealSerial(t *testing.T) {
	store, ctx := newIntegrationHostStore(t)

	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-serial-identity-1", Serial: "C02SERIAL"},
		OrbitNodeKey: "orbit-key-serial-1",
	})
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
	}); err == nil {
		t.Fatal("duplicate hardware serial insert succeeded")
	}
}

func TestNodeKeyLookupThrottlesLivenessWrites(t *testing.T) {
	store, ctx := newIntegrationHostStore(t)
	host, err := store.UpsertOnOsqueryEnroll(ctx, InventoryUpdate{
		Hardware:       HostHardware{UUID: "test-throttled-node-key-touch"},
		OsqueryNodeKey: "test-throttled-node-key-touch",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if host.Timestamps.LastSeenAt == nil {
		t.Fatal("enrolled host last_seen_at is nil")
	}

	first, err := store.GetByOsqueryNodeKey(ctx, "test-throttled-node-key-touch")
	if err != nil {
		t.Fatalf("first node key lookup: %v", err)
	}
	second, err := store.GetByOsqueryNodeKey(ctx, "test-throttled-node-key-touch")
	if err != nil {
		t.Fatalf("second node key lookup: %v", err)
	}
	if !first.Timestamps.LastSeenAt.Equal(*second.Timestamps.LastSeenAt) {
		t.Fatalf(
			"second lookup changed last_seen_at from %v to %v",
			first.Timestamps.LastSeenAt,
			second.Timestamps.LastSeenAt,
		)
	}
	if !first.Timestamps.UpdatedAt.Equal(second.Timestamps.UpdatedAt) {
		t.Fatalf(
			"liveness lookup changed updated_at from %v to %v",
			first.Timestamps.UpdatedAt,
			second.Timestamps.UpdatedAt,
		)
	}

	oldLastSeen := time.Now().Add(-2 * time.Minute)
	if _, err := store.db.Pool().
		Exec(ctx, `UPDATE hosts SET last_seen_at = $2 WHERE id = $1`, host.ID, oldLastSeen); err != nil {
		t.Fatalf("age last_seen_at: %v", err)
	}
	touched, err := store.GetByOsqueryNodeKey(ctx, "test-throttled-node-key-touch")
	if err != nil {
		t.Fatalf("aged node key lookup: %v", err)
	}
	if touched.Timestamps.LastSeenAt == nil || !touched.Timestamps.LastSeenAt.After(oldLastSeen) {
		t.Fatalf("last_seen_at = %v, want after %v", touched.Timestamps.LastSeenAt, oldLastSeen)
	}
	if !touched.Timestamps.UpdatedAt.Equal(second.Timestamps.UpdatedAt) {
		t.Fatalf(
			"liveness touch changed updated_at from %v to %v",
			second.Timestamps.UpdatedAt,
			touched.Timestamps.UpdatedAt,
		)
	}
}

func TestOrbitDeviceTokenValidationThrottlesLivenessWrites(t *testing.T) {
	store, ctx := newIntegrationHostStore(t)
	host, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-throttled-orbit-token-touch"},
		OrbitNodeKey: "test-throttled-orbit-token-touch",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	const token = "00000000-0000-4000-8000-000000000001"
	if err := store.SetOrbitDeviceAuthToken(ctx, host.OrbitNodeKey, token); err != nil {
		t.Fatalf("set device token: %v", err)
	}
	before, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host before validation: %v", err)
	}
	if err := store.ValidateOrbitDeviceAuthToken(ctx, token); err != nil {
		t.Fatalf("validate device token: %v", err)
	}
	after, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host after validation: %v", err)
	}
	if !before.Timestamps.LastSeenAt.Equal(*after.Timestamps.LastSeenAt) {
		t.Fatalf(
			"validation changed last_seen_at from %v to %v",
			before.Timestamps.LastSeenAt,
			after.Timestamps.LastSeenAt,
		)
	}
	if !before.Timestamps.UpdatedAt.Equal(after.Timestamps.UpdatedAt) {
		t.Fatalf("validation changed updated_at from %v to %v", before.Timestamps.UpdatedAt, after.Timestamps.UpdatedAt)
	}

	oldLastSeen := time.Now().Add(-2 * time.Minute)
	if _, err := store.db.Pool().
		Exec(ctx, `UPDATE hosts SET last_seen_at = $2 WHERE id = $1`, host.ID, oldLastSeen); err != nil {
		t.Fatalf("age last_seen_at: %v", err)
	}
	if err := store.ValidateOrbitDeviceAuthToken(ctx, token); err != nil {
		t.Fatalf("validate aged device token: %v", err)
	}
	touched, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get touched host: %v", err)
	}
	if touched.Timestamps.LastSeenAt == nil || !touched.Timestamps.LastSeenAt.After(oldLastSeen) {
		t.Fatalf("last_seen_at = %v, want after %v", touched.Timestamps.LastSeenAt, oldLastSeen)
	}
	if !touched.Timestamps.UpdatedAt.Equal(after.Timestamps.UpdatedAt) {
		t.Fatalf(
			"validation touch changed updated_at from %v to %v",
			after.Timestamps.UpdatedAt,
			touched.Timestamps.UpdatedAt,
		)
	}
}

func TestResolveSelectedTargetsMergesDirectHostsAndLabels(t *testing.T) {
	store, ctx := newIntegrationHostStore(t)
	labelStore := labels.NewStore(store.db)

	directHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-target-direct"},
		OrbitNodeKey: "orbit-key-direct",
	})
	if err != nil {
		t.Fatalf("enroll direct host: %v", err)
	}
	labelHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-target-label"},
		OrbitNodeKey: "orbit-key-label",
	})
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
	store, ctx := newIntegrationHostStore(t)
	labelStore := labels.NewStore(store.db)
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)

	onlineHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-count-online"},
		OrbitNodeKey: "orbit-key-count-online",
	})
	if err != nil {
		t.Fatalf("enroll online host: %v", err)
	}
	offlineHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-count-offline"},
		OrbitNodeKey: "orbit-key-count-offline",
	})
	if err != nil {
		t.Fatalf("enroll offline host: %v", err)
	}
	if _, err := store.db.Pool().Exec(ctx,
		`UPDATE hosts
		 SET last_seen_at = CASE id
		     WHEN $1 THEN $3::timestamptz
		     WHEN $2 THEN $4::timestamptz
		 END
		 WHERE id = ANY($5::bigint[])`,
		onlineHost.ID,
		offlineHost.ID,
		now.Add(-time.Minute),
		now.Add(-10*time.Minute),
		[]int64{onlineHost.ID, offlineHost.ID},
	); err != nil {
		t.Fatalf("set host seen times: %v", err)
	}
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
	store, ctx := newIntegrationHostStore(t)
	labelStore := labels.NewStore(store.db)
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)

	onlineHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-online-target-online"},
		OrbitNodeKey: "orbit-key-live-online",
	})
	if err != nil {
		t.Fatalf("enroll online host: %v", err)
	}
	offlineHost, err := store.UpsertOnOrbitEnroll(ctx, InventoryUpdate{
		Hardware:     HostHardware{UUID: "test-live-online-target-offline"},
		OrbitNodeKey: "orbit-key-live-offline",
	})
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
	if _, err := store.db.Pool().Exec(ctx,
		`UPDATE hosts
		 SET last_seen_at = CASE id
		     WHEN $1 THEN $3::timestamptz
		     WHEN $2 THEN $4::timestamptz
		 END
		 WHERE id = ANY($5::bigint[])`,
		onlineHost.ID,
		offlineHost.ID,
		now.Add(-time.Minute),
		now.Add(-10*time.Minute),
		[]int64{onlineHost.ID, offlineHost.ID},
	); err != nil {
		t.Fatalf("set host seen times: %v", err)
	}

	got, err := store.ResolveOnlineSelectedTargets(ctx, TargetSelection{
		HostIDs:  []int64{onlineHost.ID},
		LabelIDs: []int64{label.ID},
	}, now)
	if err != nil {
		t.Fatalf("resolve online selected targets: %v", err)
	}
	if !sameIDs(got, []int64{onlineHost.ID}) {
		t.Fatalf("online host ids = %v, want only online host", got)
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

func newIntegrationHostStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	database, ctx := dbtest.Open(t)
	return NewStore(database), ctx
}
