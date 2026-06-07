package software

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestSoftwareTargetsValidatePackageSelectorAndFeaturedRules(t *testing.T) {
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
				State: SoftwareStateManagedInstall,
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
				State: SoftwareStateOptionalInstall,
			},
		},
		{
			name: "specific requires package id",
			include: SoftwareInclude{
				LabelID: 1,
				Package: SoftwarePackageSelector{
					Strategy: SoftwarePackageSpecific,
				},
				State: SoftwareStateManagedInstall,
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
				State: SoftwareStateManagedInstall,
			},
			wantErr: true,
		},
		{
			name: "featured requires optional install",
			include: SoftwareInclude{
				LabelID: 1,
				Package: SoftwarePackageSelector{
					Strategy: SoftwarePackageLatest,
				},
				State:    SoftwareStateManagedInstall,
				Featured: true,
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
