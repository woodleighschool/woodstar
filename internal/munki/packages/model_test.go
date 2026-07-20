package packages

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func validPackageMutation() PackageMutation {
	installerObjectID := int64(1)
	return PackageMutation{
		Version:           "1.0",
		InstallerType:     InstallerTypePkg,
		InstallerObjectID: &installerObjectID,
	}
}

func TestPackageMutationValidateInstallerRelationship(t *testing.T) {
	t.Parallel()
	installerObjectID := int64(1)
	cases := []struct {
		name     string
		mutation PackageMutation
		wantErr  bool
	}{
		{name: "nopkg", mutation: PackageMutation{Version: "1.0", InstallerType: InstallerTypeNoPkg}},
		{
			name: "pkg",
			mutation: PackageMutation{
				Version:           "1.0",
				InstallerType:     InstallerTypePkg,
				InstallerObjectID: &installerObjectID,
			},
		},
		{
			name: "copy from dmg",
			mutation: PackageMutation{
				Version:           "1.0",
				InstallerType:     InstallerTypeCopyFromDMG,
				InstallerObjectID: &installerObjectID,
				ItemsToCopy:       []PackageItemToCopy{{SourceItem: "Example.app", DestinationPath: "/Applications"}},
			},
		},
		{
			name: "nopkg with installer",
			mutation: PackageMutation{
				Version:           "1.0",
				InstallerType:     InstallerTypeNoPkg,
				InstallerObjectID: &installerObjectID,
			},
			wantErr: true,
		},
		{
			name:     "pkg without installer",
			mutation: PackageMutation{Version: "1.0", InstallerType: InstallerTypePkg},
			wantErr:  true,
		},
		{
			name: "copy from dmg without installer",
			mutation: PackageMutation{
				Version:       "1.0",
				InstallerType: InstallerTypeCopyFromDMG,
				ItemsToCopy:   []PackageItemToCopy{{SourceItem: "Example.app", DestinationPath: "/Applications"}},
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.mutation.validate()
			if tc.wantErr && !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("validate() error = %v, want ErrInvalidInput", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validate() error = %v, want nil", err)
			}
		})
	}
}

func TestPackageMutationValidateAccepts(t *testing.T) {
	t.Parallel()
	m := validPackageMutation()
	m.SupportedArchitectures = []string{"arm64", "x86_64"}
	m.Installs = []PackageInstallItem{{Type: PackageInstallItemFile, Path: "/Applications/Foo.app"}}
	m.Receipts = []PackageReceipt{{PackageID: "com.example.foo"}}
	m.InstallerEnvironment = []PackageInstallerEnvironmentVariable{{Name: "LANG", Value: "C"}}
	m.BlockingApplications = []string{"Foo"}
	m.InstallerChoicesXML = []PackageInstallerChoice{{ChoiceIdentifier: "choice"}}
	if err := m.validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestPackageMutationValidateAllowsDisabledUninstallWithConfiguredMethod(t *testing.T) {
	t.Parallel()
	m := validPackageMutation()
	m.UninstallMethod = UninstallMethodUninstallScript
	if err := m.validate(); err != nil {
		t.Fatalf("Validate() = %v, want disabled uninstall policy to retain its method", err)
	}
}

func TestPackageMutationValidateRejects(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*PackageMutation){
		"missing version":          func(m *PackageMutation) { m.Version = "" },
		"negative installed size":  func(m *PackageMutation) { m.InstalledSize = -1 },
		"unsupported architecture": func(m *PackageMutation) { m.SupportedArchitectures = []string{"ppc"} },
		"install without path":     func(m *PackageMutation) { m.Installs = []PackageInstallItem{{Type: PackageInstallItemFile}} },
		"install blank path": func(m *PackageMutation) {
			m.Installs = []PackageInstallItem{{Type: PackageInstallItemFile, Path: "   "}}
		},
		"install unsupported type": func(m *PackageMutation) { m.Installs = []PackageInstallItem{{Type: "bogus", Path: "/x"}} },
		"receipt without id":       func(m *PackageMutation) { m.Receipts = []PackageReceipt{{}} },
		"receipt negative size": func(m *PackageMutation) {
			m.Receipts = []PackageReceipt{{PackageID: "com.example.foo", InstalledSize: -1}}
		},
		"remove copied items without items": func(m *PackageMutation) {
			m.Uninstallable = true
			m.UninstallMethod = UninstallMethodRemoveCopiedItems
		},
		"copy from dmg without items": func(m *PackageMutation) {
			m.InstallerType = InstallerTypeCopyFromDMG
		},
		"remove packages without receipts": func(m *PackageMutation) {
			m.Uninstallable = true
			m.UninstallMethod = UninstallMethodRemovePackages
		},
		"uninstall script without script": func(m *PackageMutation) {
			m.Uninstallable = true
			m.UninstallMethod = UninstallMethodUninstallScript
		},
		"uninstallable without method": func(m *PackageMutation) { m.Uninstallable = true },
		"env without name": func(m *PackageMutation) {
			m.InstallerEnvironment = []PackageInstallerEnvironmentVariable{{Value: "C"}}
		},
		"duplicate env name": func(m *PackageMutation) {
			m.InstallerEnvironment = []PackageInstallerEnvironmentVariable{
				{Name: "PATH", Value: "/usr/bin"},
				{Name: " PATH ", Value: "/usr/local/bin"},
			}
		},
		"blank blocking app": func(m *PackageMutation) { m.BlockingApplications = []string{"  "} },
		"blocking apps with none switch": func(m *PackageMutation) {
			m.BlockingApplicationsNone = true
			m.BlockingApplications = []string{"Foo"}
		},
		"choice without identifier": func(m *PackageMutation) {
			m.InstallerChoicesXML = []PackageInstallerChoice{{ChoiceAttribute: "selected"}}
		},
		"reference without software": func(m *PackageMutation) {
			m.Requires = []PackageReferenceMutation{{SoftwareID: 0}}
		},
		"reference with negative package": func(m *PackageMutation) {
			m.Requires = []PackageReferenceMutation{{SoftwareID: 1, PackageID: -1}}
		},
		"none uninstall sentinel": func(m *PackageMutation) { m.UninstallMethod = "none" },
		"none restart sentinel":   func(m *PackageMutation) { m.RestartAction = "None" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := validPackageMutation()
			mutate(&m)
			if err := m.validate(); !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("Validate() = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestPackageCreateMutationRejectsInvalidSoftwareID(t *testing.T) {
	t.Parallel()

	m := PackageCreateMutation{
		SoftwareID: -1,
		PackageMutation: PackageMutation{
			Version:       "1.0",
			InstallerType: InstallerTypeNoPkg,
		},
	}
	if err := m.validate(); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("validate() error = %v, want ErrInvalidInput", err)
	}
}
