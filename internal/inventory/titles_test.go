package inventory

import "testing"

func TestSoftwareSigningIdentifier(t *testing.T) {
	tests := []struct {
		name           string
		teamIdentifier string
		identifier     string
		want           string
	}{
		{
			name:           "compound identifier",
			teamIdentifier: "2BUA8C4S2C",
			identifier:     "com.agilebits.onepassword7",
			want:           "2BUA8C4S2C:com.agilebits.onepassword7",
		},
		{name: "missing team identifier", identifier: "com.example.app"},
		{name: "missing identifier", teamIdentifier: "TEAMID1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := softwareSigningIdentifier(tt.teamIdentifier, tt.identifier); got != tt.want {
				t.Fatalf("softwareSigningIdentifier() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSoftwareDeveloperName(t *testing.T) {
	tests := []struct {
		name           string
		teamIdentifier string
		authority      string
		want           string
	}{
		{
			name:           "developer ID authority",
			teamIdentifier: "2BUA8C4S2C",
			authority:      "Developer ID Application: AgileBits Inc. (2BUA8C4S2C)",
			want:           "AgileBits Inc.",
		},
		{
			name:           "mismatched team identifier",
			teamIdentifier: "TEAMID1234",
			authority:      "Developer ID Application: Example, Inc. (OTHERTEAM1)",
		},
		{
			name:           "authority without developer name shape",
			teamIdentifier: "TEAMID1234",
			authority:      "Example, Inc. (TEAMID1234)",
		},
		{
			name:           "authority without team identifier",
			teamIdentifier: "TEAMID1234",
			authority:      "Developer ID Application: Example, Inc.",
		},
		{name: "missing authority", teamIdentifier: "TEAMID1234"},
		{
			name:      "missing team identifier",
			authority: "Developer ID Application: Example, Inc. (TEAMID1234)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := softwareDeveloperName(tt.teamIdentifier, tt.authority); got != tt.want {
				t.Fatalf("softwareDeveloperName() = %q, want %q", got, tt.want)
			}
		})
	}
}
