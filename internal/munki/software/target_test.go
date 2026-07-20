package software

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestSoftwareTargetsValidatePackageSelectorAndActionRules(t *testing.T) {
	packageID := int64(123)
	cases := []struct {
		name    string
		include Include
		wantErr bool
	}{
		{
			name: "latest",
			include: Include{
				LabelID: 1,
				Package: PackageSelector{
					Strategy: PackageLatest,
				},
				Actions: []Action{ActionManagedInstalls},
			},
		},
		{
			name: "specific",
			include: Include{
				LabelID: 1,
				Package: PackageSelector{
					Strategy:  PackageSpecific,
					PackageID: &packageID,
				},
				Actions: []Action{ActionOptionalInstalls, ActionFeaturedItems},
			},
		},
		{
			name: "featured does not require optional install",
			include: Include{
				LabelID: 1,
				Package: PackageSelector{
					Strategy: PackageLatest,
				},
				Actions: []Action{ActionFeaturedItems},
			},
		},
		{
			name: "specific requires package id",
			include: Include{
				LabelID: 1,
				Package: PackageSelector{
					Strategy: PackageSpecific,
				},
				Actions: []Action{ActionManagedInstalls},
			},
			wantErr: true,
		},
		{
			name: "latest rejects package id",
			include: Include{
				LabelID: 1,
				Package: PackageSelector{
					Strategy:  PackageLatest,
					PackageID: &packageID,
				},
				Actions: []Action{ActionManagedInstalls},
			},
			wantErr: true,
		},
		{
			name: "actions required",
			include: Include{
				LabelID: 1,
				Package: PackageSelector{
					Strategy: PackageLatest,
				},
			},
			wantErr: true,
		},
		{
			name: "unsupported action",
			include: Include{
				LabelID: 1,
				Package: PackageSelector{
					Strategy: PackageLatest,
				},
				Actions: []Action{"managed_install"},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Targets{Include: []Include{tc.include}}.validate()
			if tc.wantErr {
				if !errors.Is(err, dbutil.ErrInvalidInput) {
					t.Fatalf("validate error = %v, want ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validate: %v", err)
			}
		})
	}
}

func TestSoftwareTargetsRejectDuplicateAndOverlappingLabels(t *testing.T) {
	t.Parallel()

	include := Include{
		LabelID: 1,
		Package: PackageSelector{Strategy: PackageLatest},
		Actions: []Action{ActionManagedInstalls},
	}
	tests := map[string]Targets{
		"duplicate include": {
			Include: []Include{include, include},
		},
		"duplicate exclude": {
			Exclude: []targeting.LabelRef{{LabelID: 2}, {LabelID: 2}},
		},
		"include and exclude overlap": {
			Include: []Include{include},
			Exclude: []targeting.LabelRef{{LabelID: 1}},
		},
	}

	for name, targets := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := targets.validate(); !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("validate error = %v, want ErrInvalidInput", err)
			}
		})
	}
}
