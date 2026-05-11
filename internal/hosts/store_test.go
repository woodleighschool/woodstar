package hosts

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestDisplayNamePriority(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   EnrollParams
		want string
	}{
		{
			name: "computer name wins",
			in: EnrollParams{
				ComputerName: "Example MacBook Pro",
				Hostname:     "example-macbook-pro",
				HardwareUUID: "uuid-1",
			},
			want: "Example MacBook Pro",
		},
		{
			name: "hostname when no computer name",
			in:   EnrollParams{Hostname: "example-macbook-pro", HardwareUUID: "uuid-1"},
			want: "example-macbook-pro",
		},
		{
			name: "uuid when no friendly name",
			in:   EnrollParams{HardwareUUID: "uuid-1"},
			want: "uuid-1",
		},
		{
			name: "whitespace-only fields fall through",
			in:   EnrollParams{ComputerName: "  ", Hostname: " ", HardwareUUID: "uuid-2"},
			want: "uuid-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := displayName(tt.in.HardwareUUID, tt.in.Hostname, tt.in.ComputerName); got != tt.want {
				t.Fatalf("displayName = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyDetailAcceptsBigPhysicalMemory(t *testing.T) {
	store, ctx := newIntegrationHostStore(t)

	host, err := store.UpsertOnOsqueryEnroll(ctx, HostDetailUpdate{
		HardwareUUID:   "test-apply-detail-big-memory",
		OsqueryNodeKey: "node-key",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	const memoryBytes = int64(68719476736)
	if err := store.ApplyDetail(ctx, host.ID, HostDetailUpdate{PhysicalMemory: memoryBytes}); err != nil {
		t.Fatalf("apply detail: %v", err)
	}

	got, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	if got.PhysicalMemory != memoryBytes {
		t.Fatalf("PhysicalMemory = %d, want %d", got.PhysicalMemory, memoryBytes)
	}
}

// Newly-enrolled hosts must land in the All Hosts builtin label so anything
// targeting it (live queries, checks, scheduled queries) sees the new host.
func TestEnrollAddsHostToAllHosts(t *testing.T) {
	store, ctx := newIntegrationHostStore(t)
	labelStore := labels.NewLabelStore(store.db)

	host, err := store.UpsertOnOrbitEnroll(ctx, EnrollParams{
		HardwareUUID: "test-enroll-all-hosts",
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

func newIntegrationHostStore(t *testing.T) (*HostStore, context.Context) {
	t.Helper()
	database, ctx := dbtest.Open(t)
	return NewHostStore(database), ctx
}

func TestCleanHostListParams(t *testing.T) {
	params := cleanHostListParams(HostListParams{
		ListParams: dbutil.ListParams{
			Q:              " mac ",
			Page:           -1,
			PerPage:        1000,
			OrderDirection: "DESC",
		},
		Status:   " online ",
		Platform: " darwin ",
		LabelID:  42,
	})

	if params.Q != "mac" {
		t.Fatalf("Q = %q, want mac", params.Q)
	}
	if params.Page != 1 {
		t.Fatalf("Page = %d, want 1", params.Page)
	}
	if params.PerPage != 200 {
		t.Fatalf("PerPage = %d, want %d", params.PerPage, 200)
	}
	if params.OrderDirection != "desc" {
		t.Fatalf("OrderDirection = %q, want desc", params.OrderDirection)
	}
	if params.Status != "online" {
		t.Fatalf("Status = %q, want online", params.Status)
	}
	if params.Platform != "darwin" {
		t.Fatalf("Platform = %q, want darwin", params.Platform)
	}
	if params.LabelID != 42 {
		t.Fatalf("LabelID = %d, want 42", params.LabelID)
	}
}
