//go:build postgres

package inventory

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestGetTitleLoadsVersionCollection(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)

	var titleID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO software_titles (name, source, bundle_identifier)
		VALUES ('Versioned App', 'apps', 'com.example.versioned')
		RETURNING id
	`).Scan(&titleID); err != nil {
		t.Fatalf("insert software title: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO software (title_id, name, version, source, bundle_identifier)
		VALUES
			($1, 'Versioned App', '2.0', 'apps', 'com.example.versioned'),
			($1, 'Versioned App', '1.0', 'apps', 'com.example.versioned')
	`, titleID); err != nil {
		t.Fatalf("insert software versions: %v", err)
	}

	title, err := store.GetTitle(ctx, titleID)
	if err != nil {
		t.Fatalf("GetTitle: %v", err)
	}
	if title.Versions.Count != 2 {
		t.Fatalf("version count = %d, want 2", title.Versions.Count)
	}
	if len(title.Versions.Items) != 2 {
		t.Fatalf("version items = %d, want 2", len(title.Versions.Items))
	}
	if got := title.Versions.Items[0].Version; got != "1.0" {
		t.Fatalf("first version = %q, want 1.0", got)
	}
	if got := title.Versions.Items[1].Version; got != "2.0" {
		t.Fatalf("second version = %q, want 2.0", got)
	}
}

func TestGetTitleLoadsSigningIdentities(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db)

	for _, hostName := range []string{"signing-host-one", "signing-host-two"} {
		host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
			Hardware:     hosts.HostHardware{UUID: hostName},
			OrbitNodeKey: hostName + "-orbit",
		})
		if err != nil {
			t.Fatalf("enroll %s: %v", hostName, err)
		}
		if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{{
			Name:             "Example",
			Version:          "1.0",
			Source:           "apps",
			BundleIdentifier: "com.example.app",
			InstalledPath:    "/Applications/Example.app",
			TeamIdentifier:   "TEAMID1234",
			Identifier:       "com.example.app",
			SigningAuthority: "Developer ID Application: Example",
		}}); err != nil {
			t.Fatalf("replace software for %s: %v", hostName, err)
		}
	}

	var titleID int64
	if err := db.Pool().QueryRow(
		ctx,
		`SELECT id FROM software_titles WHERE bundle_identifier = 'com.example.app'`,
	).Scan(&titleID); err != nil {
		t.Fatalf("get title ID: %v", err)
	}
	title, err := store.GetTitle(ctx, titleID)
	if err != nil {
		t.Fatalf("GetTitle: %v", err)
	}
	if title.SigningIdentities.Count != 1 || len(title.SigningIdentities.Items) != 1 {
		t.Fatalf("signing identities = %+v, want one identity", title.SigningIdentities)
	}
	identity := title.SigningIdentities.Items[0]
	if identity.Identifier != "com.example.app" ||
		identity.TeamIdentifier != "TEAMID1234" ||
		identity.HostsCount != 2 ||
		len(identity.Authorities) != 1 ||
		identity.Authorities[0] != "Developer ID Application: Example" {
		t.Fatalf("signing identity = %+v, want aggregated observed identity", identity)
	}
}

func TestGetTitleLoadsTeamIdentityWithoutIdentifier(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "team-only-host"},
		OrbitNodeKey: "team-only-host-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{{
		Name:             "Team Only",
		Version:          "1.0",
		Source:           "apps",
		BundleIdentifier: "com.example.team-only",
		InstalledPath:    "/Applications/Team Only.app",
		TeamIdentifier:   "TEAMID1234",
	}}); err != nil {
		t.Fatalf("replace software: %v", err)
	}

	var titleID int64
	if err := db.Pool().QueryRow(
		ctx,
		`SELECT id FROM software_titles WHERE bundle_identifier = 'com.example.team-only'`,
	).Scan(&titleID); err != nil {
		t.Fatalf("get title ID: %v", err)
	}
	title, err := store.GetTitle(ctx, titleID)
	if err != nil {
		t.Fatalf("GetTitle: %v", err)
	}
	if title.SigningIdentities.Count != 1 || len(title.SigningIdentities.Items) != 1 {
		t.Fatalf("signing identities = %+v, want one identity", title.SigningIdentities)
	}
	identity := title.SigningIdentities.Items[0]
	if identity.TeamIdentifier != "TEAMID1234" ||
		identity.Identifier != "" {
		t.Fatalf("signing identity = %+v, want Team ID without Identifier", identity)
	}
}
