package packages

import "testing"

func TestMunkiPackageFromDomainFallsBackToSoftwareIconURL(t *testing.T) {
	softwareIconID := int64(41)
	packageIconID := int64(42)

	tests := []struct {
		name         string
		pkg          Package
		wantIconURL  string
		wantOverride *int64
	}{
		{
			name:        "inherited software icon",
			pkg:         Package{SoftwareIconArtifactID: &softwareIconID},
			wantIconURL: "/api/munki/artifacts/41/content",
		},
		{
			name: "package icon override",
			pkg: Package{
				SoftwareIconArtifactID: &softwareIconID,
				IconArtifactID:         &packageIconID,
			},
			wantIconURL:  "/api/munki/artifacts/42/content",
			wantOverride: &packageIconID,
		},
		{
			name:        "no icon",
			pkg:         Package{},
			wantIconURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := munkiPackageFromDomain(tt.pkg)
			if got.IconURL != tt.wantIconURL {
				t.Fatalf("icon url = %q, want %q", got.IconURL, tt.wantIconURL)
			}
			if got.IconArtifactID != tt.wantOverride {
				t.Fatalf("package icon artifact id = %v, want %v", got.IconArtifactID, tt.wantOverride)
			}
		})
	}
}
