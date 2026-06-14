package packages

import "testing"

func TestMunkiPackageFromPackageUsesIconURL(t *testing.T) {
	softwareIconID := int64(41)

	tests := []struct {
		name        string
		pkg         Package
		wantIconURL string
	}{
		{
			name:        "software icon",
			pkg:         Package{IconObjectID: &softwareIconID},
			wantIconURL: "/api/munki/icons/41/content",
		},
		{
			name:        "no icon",
			pkg:         Package{},
			wantIconURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MunkiPackageFromPackage(tt.pkg)
			if got.IconURL != tt.wantIconURL {
				t.Fatalf("icon url = %q, want %q", got.IconURL, tt.wantIconURL)
			}
		})
	}
}
