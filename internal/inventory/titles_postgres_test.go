//go:build postgres

package inventory

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestGetTitleLoadsVersionCollection(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)

	var titleID int64
	if err := db.QueryRow(ctx, `
		INSERT INTO software_titles (name, source, bundle_identifier)
		VALUES ('Versioned App', 'apps', 'com.example.versioned')
		RETURNING id
	`).Scan(&titleID); err != nil {
		t.Fatalf("insert software title: %v", err)
	}
	if _, err := db.Exec(ctx, `
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
	hostStore := hosts.NewStore(db, labels.NewStore(db))

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
			Signature: &SoftwareCodeSignature{
				Signed:         true,
				TeamIdentifier: "TEAMID1234",
				Identifier:     "com.example.app",
				Authority:      "Developer ID Application: Example, Inc. (TEAMID1234)",
			},
		}}); err != nil {
			t.Fatalf("replace software for %s: %v", hostName, err)
		}
	}

	var titleID int64
	if err := db.QueryRow(
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
		identity.SigningIdentifier != "TEAMID1234:com.example.app" ||
		identity.TeamIdentifier != "TEAMID1234" ||
		identity.DeveloperName != "Example, Inc." ||
		identity.HostsCount != 2 ||
		identity.Authority != "Developer ID Application: Example, Inc. (TEAMID1234)" {
		t.Fatalf("signing identity = %+v, want aggregated observed identity", identity)
	}

	params := SoftwareTitleListParams{}
	params.ListParams.Q = "Example, Inc."
	titles, total, err := store.ListTitles(ctx, params)
	if err != nil {
		t.Fatalf("ListTitles by developer name: %v", err)
	}
	if total != 1 || len(titles) != 1 || titles[0].ID != titleID {
		t.Fatalf("developer name search = (%d, %+v), want title %d", total, titles, titleID)
	}
}

func TestGetTitleSeparatesAuthoritiesAndExcludesUnsignedSignatures(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db, labels.NewStore(db))

	observations := []struct {
		host      string
		signature *SoftwareCodeSignature
	}{
		{
			host: "authority-one",
			signature: &SoftwareCodeSignature{
				Signed:         true,
				Identifier:     "com.example.rotated",
				TeamIdentifier: "TEAMID1234",
				Authority:      "Developer ID Application: Example One (TEAMID1234)",
			},
		},
		{
			host: "authority-two",
			signature: &SoftwareCodeSignature{
				Signed:         true,
				Identifier:     "com.example.rotated",
				TeamIdentifier: "TEAMID1234",
				Authority:      "Developer ID Application: Example Two (TEAMID1234)",
			},
		},
		{
			host: "unsigned-signature",
			signature: &SoftwareCodeSignature{
				Identifier:     "com.example.rotated",
				TeamIdentifier: "TEAMID1234",
				Authority:      "Developer ID Application: Unsigned (TEAMID1234)",
			},
		},
		{host: "unobserved-signature"},
	}

	for _, observation := range observations {
		host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
			Hardware:     hosts.HostHardware{UUID: observation.host},
			OrbitNodeKey: observation.host + "-orbit",
		})
		if err != nil {
			t.Fatalf("enroll %s: %v", observation.host, err)
		}
		if err := store.ReplaceHostSoftware(ctx, host.ID, []HostSoftwareEntry{{
			Name:             "Rotated App",
			Version:          "1.0",
			Source:           "apps",
			BundleIdentifier: "com.example.rotated",
			InstalledPath:    "/Applications/Rotated App.app",
			Signature:        observation.signature,
		}}); err != nil {
			t.Fatalf("replace software for %s: %v", observation.host, err)
		}
	}

	var titleID int64
	if err := db.QueryRow(
		ctx,
		`SELECT id FROM software_titles WHERE bundle_identifier = 'com.example.rotated'`,
	).Scan(&titleID); err != nil {
		t.Fatalf("get title ID: %v", err)
	}
	title, err := store.GetTitle(ctx, titleID)
	if err != nil {
		t.Fatalf("GetTitle: %v", err)
	}
	if title.SigningIdentities.Count != 2 || len(title.SigningIdentities.Items) != 2 {
		t.Fatalf("signing identities = %+v, want one row per signed authority", title.SigningIdentities)
	}
	if title.SigningIdentities.Items[0].DeveloperName != "Example One" ||
		title.SigningIdentities.Items[1].DeveloperName != "Example Two" {
		t.Fatalf("signing identities = %+v, want separate authority names", title.SigningIdentities)
	}

	params := SoftwareTitleListParams{}
	params.ListParams.Q = "Unsigned"
	titles, total, err := store.ListTitles(ctx, params)
	if err != nil {
		t.Fatalf("ListTitles by unsigned authority: %v", err)
	}
	if total != 0 || len(titles) != 0 {
		t.Fatalf("unsigned authority search = (%d, %+v), want no titles", total, titles)
	}
}

func TestGetTitleLoadsTeamIdentityWithoutIdentifier(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	hostStore := hosts.NewStore(db, labels.NewStore(db))

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
		Signature: &SoftwareCodeSignature{
			Signed:         true,
			TeamIdentifier: "TEAMID1234",
		},
	}}); err != nil {
		t.Fatalf("replace software: %v", err)
	}

	var titleID int64
	if err := db.QueryRow(
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
		identity.Identifier != "" ||
		identity.SigningIdentifier != "" ||
		identity.DeveloperName != "" {
		t.Fatalf("signing identity = %+v, want Team ID without Identifier", identity)
	}
}
