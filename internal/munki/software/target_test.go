package software

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestSoftwareTargetsValidatePackageSelectorAndActionRules(t *testing.T) {
	packageID := int64(123)
	cases := []struct {
		name    string
		include SoftwareInclude
		wantErr bool
	}{
		{
			name: "latest",
			include: SoftwareInclude{
				LabelID: 1,
				Package: SoftwarePackageSelector{
					Strategy: SoftwarePackageLatest,
				},
				Actions: []SoftwareAction{SoftwareActionManagedInstalls},
			},
		},
		{
			name: "specific",
			include: SoftwareInclude{
				LabelID: 1,
				Package: SoftwarePackageSelector{
					Strategy:  SoftwarePackageSpecific,
					PackageID: &packageID,
				},
				Actions: []SoftwareAction{SoftwareActionOptionalInstalls, SoftwareActionFeaturedItems},
			},
		},
		{
			name: "featured does not require optional install",
			include: SoftwareInclude{
				LabelID: 1,
				Package: SoftwarePackageSelector{
					Strategy: SoftwarePackageLatest,
				},
				Actions: []SoftwareAction{SoftwareActionFeaturedItems},
			},
		},
		{
			name: "specific requires package id",
			include: SoftwareInclude{
				LabelID: 1,
				Package: SoftwarePackageSelector{
					Strategy: SoftwarePackageSpecific,
				},
				Actions: []SoftwareAction{SoftwareActionManagedInstalls},
			},
			wantErr: true,
		},
		{
			name: "latest rejects package id",
			include: SoftwareInclude{
				LabelID: 1,
				Package: SoftwarePackageSelector{
					Strategy:  SoftwarePackageLatest,
					PackageID: &packageID,
				},
				Actions: []SoftwareAction{SoftwareActionManagedInstalls},
			},
			wantErr: true,
		},
		{
			name: "actions required",
			include: SoftwareInclude{
				LabelID: 1,
				Package: SoftwarePackageSelector{
					Strategy: SoftwarePackageLatest,
				},
			},
			wantErr: true,
		},
		{
			name: "unsupported action",
			include: SoftwareInclude{
				LabelID: 1,
				Package: SoftwarePackageSelector{
					Strategy: SoftwarePackageLatest,
				},
				Actions: []SoftwareAction{"managed_install"},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := SoftwareTargets{Include: []SoftwareInclude{tc.include}}.validate()
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
