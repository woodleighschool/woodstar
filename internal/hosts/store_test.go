package hosts

import (
	"context"
	"errors"
	"strings"
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

func TestHostListWhereCheckFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		params      ListParams
		wantSQL     string
		wantErr     bool
		wantArgLen  int
		wantCheckID int64
	}{
		{
			name:        "pass",
			params:      ListParams{CheckID: 42, CheckResponse: CheckResponsePass},
			wantSQL:     "cm.passes IS TRUE",
			wantArgLen:  1,
			wantCheckID: 42,
		},
		{
			name:        "fail",
			params:      ListParams{CheckID: 42, CheckResponse: CheckResponseFail},
			wantSQL:     "cm.passes IS FALSE",
			wantArgLen:  1,
			wantCheckID: 42,
		},
		{
			name:    "missing response",
			params:  ListParams{CheckID: 42},
			wantErr: true,
		},
		{
			name:    "missing check",
			params:  ListParams{CheckResponse: CheckResponsePass},
			wantErr: true,
		},
		{
			name:    "unknown response",
			params:  ListParams{CheckID: 42, CheckResponse: CheckResponse("passing")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			whereSQL, args, err := hostListWhere(tt.params)
			if tt.wantErr {
				if !errors.Is(err, dbutil.ErrInvalidInput) {
					t.Fatalf("hostListWhere error = %v, want invalid input", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("hostListWhere: %v", err)
			}
			if !strings.Contains(whereSQL, tt.wantSQL) {
				t.Fatalf("where SQL = %q, want %q", whereSQL, tt.wantSQL)
			}
			if len(args) != tt.wantArgLen {
				t.Fatalf("args length = %d, want %d", len(args), tt.wantArgLen)
			}
			if got := args[0]; got != tt.wantCheckID {
				t.Fatalf("check arg = %v, want %d", got, tt.wantCheckID)
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
		if l.Name == "All Hosts" &&
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
	label, err := labelStore.Create(ctx, labels.LabelCreate{
		Name:                "Live Target Test",
		LabelType:           labels.LabelTypeRegular,
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
	label, err := labelStore.Create(ctx, labels.LabelCreate{
		Name:                "Live Count Test",
		LabelType:           labels.LabelTypeRegular,
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
	label, err := labelStore.Create(ctx, labels.LabelCreate{
		Name:                "Live Online Target Test",
		LabelType:           labels.LabelTypeRegular,
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
