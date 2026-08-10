//go:build postgres

package inventory

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestListForHostMissingHost(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)

	_, _, err := store.ListForHost(ctx, 999999, HostSoftwareListParams{})
	if !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("ListForHost missing host error = %v, want ErrNotFound", err)
	}
}

func TestListForHostEmptyHost(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db, labels.NewStore(db))
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

func TestListForHostReturnsPathOwnedSignatureObservations(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	store := NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "inventory-signature-host"},
		OrbitNodeKey: "inventory-signature-host-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	base := HostSoftwareEntry{
		Name:             "Example",
		Version:          "1.0",
		Source:           "apps",
		BundleIdentifier: "com.example.app",
	}
	valid := base
	valid.InstalledPath = "/Applications/Example.app"
	valid.ExecutablePath = "/Applications/Example.app/Contents/MacOS/Example"
	valid.ExecutableSHA256 = "executable-hash"
	valid.Signature = &SoftwareCodeSignature{
		Valid:          true,
		Identifier:     "com.example.app",
		Authority:      "Developer ID Application: Example (TEAMID1234)",
		TeamIdentifier: "TEAMID1234",
		CDHash:         "valid-cdhash",
	}
	invalid := base
	invalid.InstalledPath = "/Applications/Invalid.app"
	invalid.Signature = &SoftwareCodeSignature{CDHash: "invalid-cdhash"}
	unobserved := base
	unobserved.InstalledPath = "/Applications/Unobserved.app"

	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{valid, invalid, unobserved}); err != nil {
		t.Fatalf("replace host software: %v", err)
	}
	software, total, err := store.ListForHost(ctx, host.ID, HostSoftwareListParams{})
	if err != nil {
		t.Fatalf("ListForHost: %v", err)
	}
	if total != 1 || len(software) != 1 || len(software[0].InstalledVersions) != 1 {
		t.Fatalf("host software = (%d, %+v), want one title and version", total, software)
	}
	paths := software[0].InstalledVersions[0].Paths
	if len(paths) != 3 {
		t.Fatalf("paths = %+v, want three installed paths", paths)
	}
	if paths[0].Signature == nil || !paths[0].Signature.Valid ||
		paths[0].Signature.CDHash != "valid-cdhash" ||
		paths[0].ExecutableSHA256 != "executable-hash" {
		t.Fatalf("valid path = %+v, want signature and executable observation", paths[0])
	}
	if paths[1].Signature == nil || paths[1].Signature.Valid ||
		paths[1].Signature.CDHash != "invalid-cdhash" {
		t.Fatalf("invalid path = %+v, want explicit invalid signature", paths[1])
	}
	if paths[2].Signature != nil {
		t.Fatalf("unobserved path = %+v, want nil signature", paths[2])
	}
}
