package scope

import "testing"

func TestNormalizeLabelScope(t *testing.T) {
	s := NormalizeLabelScope(LabelScope{
		Mode:     ScopeExcludeAny,
		LabelIDs: []int64{5, 2, 5, 0, -1},
	})
	if s.Mode != ScopeExcludeAny {
		t.Fatalf("Mode = %q, want %q", s.Mode, ScopeExcludeAny)
	}
	assertInt64s(t, "LabelIDs", s.LabelIDs, []int64{2, 5})

	empty := NormalizeLabelScope(LabelScope{Mode: ScopeIncludeAll})
	if empty.Mode != ScopeNone {
		t.Fatalf("empty Mode = %q, want %q", empty.Mode, ScopeNone)
	}
}

func TestPlatformFromOsquery(t *testing.T) {
	tests := []struct {
		name         string
		platform     string
		platformLike string
		want         Platform
	}{
		{name: "darwin", platform: "darwin", want: PlatformDarwin},
		{name: "legacy macos", platform: "macos", want: PlatformDarwin},
		{name: "windows", platform: "windows", want: PlatformWindows},
		{name: "ubuntu", platform: "ubuntu", want: PlatformLinux},
		{name: "rhel by platform_like", platform: "custom-linux", platformLike: "rhel", want: PlatformLinux},
		{name: "unknown is not linux", platform: "chrome", want: PlatformUnknown},
		{name: "empty", want: PlatformUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PlatformFromOsquery(tt.platform, tt.platformLike); got != tt.want {
				t.Fatalf("PlatformFromOsquery(%q, %q) = %q, want %q", tt.platform, tt.platformLike, got, tt.want)
			}
		})
	}
}

func assertInt64s(t *testing.T, name string, got []int64, want []int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}
