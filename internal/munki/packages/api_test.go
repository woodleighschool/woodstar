package packages

import "testing"

func TestMunkiPackageFromRecordUsesSoftwareIconURL(t *testing.T) {
	softwareIconID := int64(41)

	tests := []struct {
		name        string
		record      PackageRecord
		wantIconURL string
	}{
		{
			name:        "software icon",
			record:      PackageRecord{SoftwareIcon: IconRef{ObjectID: &softwareIconID}},
			wantIconURL: "/api/storage/objects/41/content",
		},
		{
			name:        "no icon",
			record:      PackageRecord{},
			wantIconURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MunkiPackageFromRecord(tt.record)
			if got.IconURL != tt.wantIconURL {
				t.Fatalf("icon url = %q, want %q", got.IconURL, tt.wantIconURL)
			}
		})
	}
}
