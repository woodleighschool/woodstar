package dbutil

import "testing"

func TestSameInt64Set(t *testing.T) {
	tests := []struct {
		name string
		a    []int64
		b    []int64
		want bool
	}{
		{name: "same order", a: []int64{1, 2, 3}, b: []int64{1, 2, 3}, want: true},
		{name: "different order", a: []int64{1, 2, 3}, b: []int64{3, 1, 2}, want: true},
		{name: "different length", a: []int64{1, 2}, b: []int64{1, 2, 3}},
		{name: "different values", a: []int64{1, 2, 3}, b: []int64{1, 2, 4}},
		{name: "duplicate does not hide missing value", a: []int64{1, 2, 3}, b: []int64{1, 1, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SameInt64Set(tt.a, tt.b); got != tt.want {
				t.Fatalf("SameInt64Set(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
