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
