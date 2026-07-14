package inventory

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

func TestListForHostMissingHost(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)

	_, _, err := store.ListForHost(ctx, 999999, HostSoftwareListParams{})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("ListForHost missing host error = %v, want ErrNotFound", err)
	}
}

func TestListForHostEmptyHost(t *testing.T) {
	db, ctx := dbtest.Open(t)
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

func TestBuildHostSoftwaresInitializesPathCollections(t *testing.T) {
	software := buildHostSoftwares([]hostSoftwareScanRow{
		{TitleID: 1, TitleName: "No Paths", SoftwareID: 2, Version: "1.0"},
	})
	if len(software) != 1 || len(software[0].InstalledVersions) != 1 {
		t.Fatalf("software = %+v, want one title and version", software)
	}
	version := software[0].InstalledVersions[0]
	if version.InstalledPaths == nil {
		t.Fatal("InstalledPaths is nil, want empty array")
	}
	if version.SignatureInformation == nil {
		t.Fatal("SignatureInformation is nil, want empty array")
	}
}
