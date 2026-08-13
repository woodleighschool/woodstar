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
	if version.Paths == nil {
		t.Fatal("Paths is nil, want empty array")
	}
}

func TestBuildHostSoftwaresExposesExactSigningIdentity(t *testing.T) {
	signed := true
	software := buildHostSoftwares([]hostSoftwareScanRow{{
		TitleID:          1,
		TitleName:        "Example",
		SoftwareID:       2,
		Version:          "1.0",
		InstalledPath:    "/Applications/Example.app",
		SignatureSigned:  &signed,
		TeamIdentifier:   "TEAMID1234",
		Identifier:       "com.example.app",
		SigningAuthority: "Developer ID Application: Example",
		CDHash:           "cdhash",
	}})

	got := software[0].InstalledVersions[0].Paths[0]
	if got.Signature == nil || !got.Signature.Signed ||
		got.Signature.Identifier != "com.example.app" ||
		got.Signature.Authority != "Developer ID Application: Example" ||
		got.Signature.CDHash != "cdhash" {
		t.Fatalf("installed path = %+v, want exact signing identity", got)
	}
}
