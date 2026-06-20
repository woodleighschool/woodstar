package packages

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func validPackageMutation() PackageMutation {
	return PackageMutation{
		Version:       "1.0",
		InstallerType: InstallerTypePkg,
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
	if err := m.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestPackageMutationValidateRejects(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*PackageMutation){
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
			m.UninstallMethod = UninstallMethodRemoveCopiedItems
		},
		"env without name":   func(m *PackageMutation) { m.InstallerEnvironment = []PackageInstallerEnvironmentVariable{{Value: "C"}} },
		"blank blocking app": func(m *PackageMutation) { m.BlockingApplications = []string{"  "} },
		"blocking apps with none switch": func(m *PackageMutation) {
			m.BlockingApplicationsNone = true
			m.BlockingApplications = []string{"Foo"}
		},
		"choice without identifier": func(m *PackageMutation) {
			m.InstallerChoicesXML = []PackageInstallerChoice{{ChoiceAttribute: "selected"}}
		},
		"reference without software": func(m *PackageMutation) { m.Requires = []PackageReference{{SoftwareID: 0}} },
		"none uninstall sentinel":    func(m *PackageMutation) { m.UninstallMethod = "none" },
		"none restart sentinel":      func(m *PackageMutation) { m.RestartAction = "None" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := validPackageMutation()
			mutate(&m)
			if err := m.Validate(); !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("Validate() = %v, want ErrInvalidInput", err)
			}
		})
	}
}
