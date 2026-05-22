package platforms

import "testing"

func TestCleanTargetsNormalizesAndDeduplicates(t *testing.T) {
	got, err := CleanTargets([]Platform{" darwin ", "windows", "DARWIN", " linux "})
	if err != nil {
		t.Fatalf("CleanTargets returned error: %v", err)
	}
	assertPlatforms(t, got, []Platform{PlatformDarwin, PlatformWindows, PlatformLinux})
}

func TestCleanTargetsRejectsInvalidTargets(t *testing.T) {
	tests := []struct {
		name string
		in   []Platform
	}{
		{name: "empty input", in: nil},
		{name: "blank target", in: []Platform{" "}},
		{name: "unknown target", in: []Platform{"darwin", "chrome"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := CleanTargets(tt.in); err == nil {
				t.Fatalf("CleanTargets(%#v) returned nil error, want invalid target error", tt.in)
			}
		})
	}
}

func assertPlatforms(t *testing.T, got []Platform, want []Platform) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("platforms = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("platforms = %#v, want %#v", got, want)
		}
	}
}
