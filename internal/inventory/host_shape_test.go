package inventory

import "testing"

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

func TestBuildHostSoftwaresExposesExactSigningIdentity(t *testing.T) {
	software := buildHostSoftwares([]hostSoftwareScanRow{{
		TitleID:          1,
		TitleName:        "Example",
		SoftwareID:       2,
		Version:          "1.0",
		InstalledPath:    "/Applications/Example.app",
		TeamIdentifier:   "TEAMID1234",
		Identifier:       "com.example.app",
		SigningAuthority: "Developer ID Application: Example",
	}})

	got := software[0].InstalledVersions[0].SignatureInformation[0]
	if got.Identifier != "com.example.app" ||
		got.SigningAuthority != "Developer ID Application: Example" {
		t.Fatalf("signature information = %+v, want exact signing identity", got)
	}
}
