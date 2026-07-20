//go:build postgres

package inventory

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestListForHostMissingHost(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)

	_, _, err := store.ListForHost(ctx, 999999, HostSoftwareListParams{})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("ListForHost missing host error = %v, want ErrNotFound", err)
	}
}

func TestListForHostEmptyHost(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	store := NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "inventory-empty-host"},
		OrbitNodeKey: "inventory-empty-host-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	rows, count, err := store.ListForHost(ctx, host.ID, HostSoftwareListParams{})
	if err != nil {
		t.Fatalf("ListForHost empty host: %v", err)
	}
	if len(rows) != 0 || count != 0 {
		t.Fatalf("ListForHost empty host = %d rows count %d, want empty page", len(rows), count)
	}
}
