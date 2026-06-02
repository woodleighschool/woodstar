package scope

import "testing"

func TestValidTargetLabelEffect(t *testing.T) {
	cases := []struct {
		name   string
		effect TargetLabelEffect
		want   bool
	}{
		{name: "include", effect: TargetLabelInclude, want: true},
		{name: "exclude", effect: TargetLabelExclude, want: true},
		{name: "empty", effect: "", want: false},
		{name: "bogus", effect: "bogus", want: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidTargetLabelEffect(tt.effect); got != tt.want {
				t.Fatalf("ValidTargetLabelEffect(%q) = %t, want %t", tt.effect, got, tt.want)
			}
		})
	}
}
