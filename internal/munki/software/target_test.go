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
			name: "featured requires optional install",
			include: Include{
				LabelID: 1,
				Package: PackageSelector{
					Strategy: PackageLatest,
				},
				Actions: []Action{ActionFeaturedItems},
			},
			wantErr: true,
		},
		{
			name:    "managed updates may combine with optional installs",
			include: Include{LabelID: 1, Package: PackageSelector{Strategy: PackageLatest}, Actions: []Action{ActionManagedUpdates, ActionOptionalInstalls}},
		},
		{
			name:    "managed updates may combine with optional installs and featured items",
			include: Include{LabelID: 1, Package: PackageSelector{Strategy: PackageLatest}, Actions: []Action{ActionManagedUpdates, ActionOptionalInstalls, ActionFeaturedItems}},
		},
		{
			name:    "managed updates may combine with optional installs and default installs",
			include: Include{LabelID: 1, Package: PackageSelector{Strategy: PackageLatest}, Actions: []Action{ActionManagedUpdates, ActionOptionalInstalls, ActionDefaultInstalls}},
		},
		{
			name:    "managed updates may combine with every optional presentation modifier",
			include: Include{LabelID: 1, Package: PackageSelector{Strategy: PackageLatest}, Actions: []Action{ActionManagedUpdates, ActionOptionalInstalls, ActionFeaturedItems, ActionDefaultInstalls}},
		},
		{
			name:    "optional installs may combine with featured and default",
			include: Include{LabelID: 1, Package: PackageSelector{Strategy: PackageLatest}, Actions: []Action{ActionOptionalInstalls, ActionFeaturedItems, ActionDefaultInstalls}},
		},
		{
			name:    "managed installs are exclusive",
			include: Include{LabelID: 1, Package: PackageSelector{Strategy: PackageLatest}, Actions: []Action{ActionManagedInstalls, ActionOptionalInstalls}},
			wantErr: true,
		},
		{
			name:    "managed uninstalls are exclusive",
			include: Include{LabelID: 1, Package: PackageSelector{Strategy: PackageLatest}, Actions: []Action{ActionManagedUninstalls, ActionManagedUpdates}},
			wantErr: true,
		},
		{
			name:    "managed updates require optional installs when combined",
			include: Include{LabelID: 1, Package: PackageSelector{Strategy: PackageLatest}, Actions: []Action{ActionManagedUpdates, ActionFeaturedItems}},
			wantErr: true,
		},
		{
			name:    "default requires optional installs",
			include: Include{LabelID: 1, Package: PackageSelector{Strategy: PackageLatest}, Actions: []Action{ActionDefaultInstalls}},
			wantErr: true,
		},
		{
			name:    "duplicate actions rejected",
			include: Include{LabelID: 1, Package: PackageSelector{Strategy: PackageLatest}, Actions: []Action{ActionOptionalInstalls, ActionOptionalInstalls}},
			wantErr: true,
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
