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
	hostStore := hosts.NewStore(db, labels.NewStore(db))
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
	signed := base
	signed.InstalledPath = "/Applications/Example.app"
	signed.ExecutablePath = "/Applications/Example.app/Contents/MacOS/Example"
	signed.ExecutableSHA256 = "executable-hash"
	signed.Signature = &SoftwareCodeSignature{
		Signed:         true,
		Identifier:     "com.example.app",
		Authority:      "Developer ID Application: Example (TEAMID1234)",
		TeamIdentifier: "TEAMID1234",
		CDHash:         "signed-cdhash",
	}
	unsigned := base
	unsigned.InstalledPath = "/Applications/Unsigned.app"
	unsigned.Signature = &SoftwareCodeSignature{CDHash: "unsigned-cdhash"}
	unobserved := base
	unobserved.InstalledPath = "/Applications/Unobserved.app"

	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{signed, unsigned, unobserved}); err != nil {
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
	pathsByName := make(map[string]SoftwareInstalledPath, len(paths))
	for _, path := range paths {
		pathsByName[path.Path] = path
	}
	signedPath := pathsByName["/Applications/Example.app"]
	if signedPath.Signature == nil || !signedPath.Signature.Signed ||
		signedPath.Signature.CDHash != "signed-cdhash" ||
		signedPath.ExecutableSHA256 != "executable-hash" {
		t.Fatalf("signed path = %+v, want signature and executable observation", signedPath)
	}
	unsignedPath := pathsByName["/Applications/Unsigned.app"]
	if unsignedPath.Signature == nil || unsignedPath.Signature.Signed ||
		unsignedPath.Signature.CDHash != "unsigned-cdhash" {
		t.Fatalf("unsigned path = %+v, want explicit unsigned signature", unsignedPath)
	}
	unobservedPath, ok := pathsByName["/Applications/Unobserved.app"]
	if !ok {
		t.Fatal("unobserved path is missing")
	}
	if unobservedPath.Signature != nil {
		t.Fatalf("unobserved path = %+v, want nil signature", unobservedPath)
	}
}
