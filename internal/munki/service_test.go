package munki_test

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

func TestResolvePackageArtifactRequiresEffectivePackage(t *testing.T) {
	installerID := int64(42)
	store := servicePackageStore{
		packages: []munkisoftware.EffectivePackage{
			{
				TargetID:   1,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package: packages.Package{
					ID:                      10,
					SoftwareName:            "GoogleChrome",
					InstallerObjectID:       &installerID,
					InstallerObjectLocation: "munki/packages/42/GoogleChrome.pkg",
				},
			},
		},
	}
	service := munki.NewRepositoryService(munki.Dependencies{Packages: store})
	client := munki.ClientHost{ID: 1, Serial: "C02MUNKI"}

	key, err := service.ResolvePackageArtifact(context.Background(), client, "munki/packages/42/GoogleChrome.pkg")
	if err != nil {
		t.Fatalf("ResolvePackageArtifact allowed package: %v", err)
	}
	if key != "munki/packages/42/GoogleChrome.pkg" {
		t.Fatalf("key = %q, want the installer key", key)
	}

	_, err = service.ResolvePackageArtifact(context.Background(), client, "munki/packages/99/Blocked.pkg")
	if !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("blocked key error = %v, want ErrNotFound", err)
	}
}

type servicePackageStore struct {
	packages []munkisoftware.EffectivePackage
}

func (s servicePackageStore) EffectivePackagesForHost(
	_ context.Context,
	hostID int64,
) ([]munkisoftware.EffectivePackage, error) {
	if hostID != 1 {
		return nil, nil
	}
	return s.packages, nil
}
