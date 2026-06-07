package targeting_test

import (
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/targeting"
)

type includeTarget struct {
	Name    string
	LabelID int64
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		includes    []includeTarget
		excludes    []targeting.LabelRef
		hostLabels  []int64
		wantMatched bool
		wantExclude bool
		wantInclude string
	}{
		{
			name:        "exclude wins",
			includes:    []includeTarget{{Name: "matching include", LabelID: 10}},
			excludes:    labelRefs(20),
			hostLabels:  []int64{10, 20},
			wantExclude: true,
		},
		{
			name: "first include wins",
			includes: []includeTarget{
				{Name: "first", LabelID: 20},
				{Name: "second", LabelID: 30},
			},
			hostLabels:  []int64{30, 20},
			wantMatched: true,
			wantInclude: "first",
		},
		{
			name:       "no include means no match",
			excludes:   labelRefs(40),
			hostLabels: []int64{50},
		},
		{
			name:       "includes present but none matching",
			includes:   []includeTarget{{Name: "unmatched", LabelID: 40}},
			hostLabels: []int64{50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := targeting.Resolve(tt.includes, tt.excludes, tt.hostLabels, includeTargetLabelID)

			if got.Matched != tt.wantMatched {
				t.Fatalf("Matched = %t, want %t", got.Matched, tt.wantMatched)
			}
			if got.Excluded != tt.wantExclude {
				t.Fatalf("Excluded = %t, want %t", got.Excluded, tt.wantExclude)
			}
			if tt.wantInclude == "" {
				if got.Include != (includeTarget{}) {
					t.Fatalf("Include = %+v, want zero value", got.Include)
				}
				return
			}
			if got.Include.Name != tt.wantInclude {
				t.Fatalf("Include = %+v, want %q", got.Include, tt.wantInclude)
			}
		})
	}
}

func TestValidateTargets(t *testing.T) {
	tests := []struct {
		name     string
		includes []includeTarget
		excludes []targeting.LabelRef
		wantErr  string
	}{
		{
			name:     "valid",
			includes: includeTargets(1, 2),
			excludes: labelRefs(3, 4),
		},
		{
			name:     "duplicate include rejected",
			includes: includeTargets(1, 1),
			wantErr:  "duplicate include label_id 1",
		},
		{
			name:     "duplicate exclude rejected",
			excludes: labelRefs(2, 2),
			wantErr:  "duplicate exclude label_id 2",
		},
		{
			name:     "same label in include and exclude rejected",
			includes: includeTargets(3),
			excludes: labelRefs(3),
			wantErr:  "label_id 3 is both included and excluded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := targeting.ValidateTargets(tt.includes, tt.excludes, includeTargetLabelID)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateTargets() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateTargets() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateTargets() error = %q, want to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLabelSets(t *testing.T) {
	tests := []struct {
		name     string
		includes []targeting.LabelRef
		excludes []targeting.LabelRef
		wantErr  string
	}{
		{
			name:     "valid label refs",
			includes: labelRefs(1),
			excludes: labelRefs(2),
		},
		{
			name:     "duplicate label ref include rejected",
			includes: labelRefs(1, 1),
			wantErr:  "duplicate include label_id 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := targeting.ValidateLabelSets(tt.includes, tt.excludes)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateLabelSets() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateLabelSets() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateLabelSets() error = %q, want to contain %q", err, tt.wantErr)
			}
		})
	}
}

func includeTargetLabelID(target includeTarget) int64 {
	return target.LabelID
}

func includeTargets(labelIDs ...int64) []includeTarget {
	targets := make([]includeTarget, len(labelIDs))
	for i, labelID := range labelIDs {
		targets[i] = includeTarget{LabelID: labelID}
	}
	return targets
}

func labelRefs(labelIDs ...int64) []targeting.LabelRef {
	refs := make([]targeting.LabelRef, len(labelIDs))
	for i, labelID := range labelIDs {
		refs[i] = targeting.LabelRef{LabelID: labelID}
	}
	return refs
}
