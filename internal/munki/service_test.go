package munki_test

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
	"github.com/woodleighschool/woodstar/internal/munki/assignments"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

func TestServiceArtifactRedirectRequiresEffectivePackage(t *testing.T) {
	artifactID := int64(42)
	store := serviceArtifactStore{
		artifacts: map[string]artifacts.Artifact{
			"package/apps/GoogleChrome.pkg": {
				ID:         artifactID,
				Kind:       artifacts.ArtifactKindPackage,
				Location:   "apps/GoogleChrome.pkg",
				StorageKey: "apps/GoogleChrome.pkg",
			},
			"package/apps/Blocked.pkg": {
				ID:         43,
				Kind:       artifacts.ArtifactKindPackage,
				Location:   "apps/Blocked.pkg",
				StorageKey: "apps/Blocked.pkg",
			},
		},
		packages: []assignments.EffectivePackage{
			{
				AssignmentID:     1,
				SoftwareID:       1,
				Action:           assignments.AssignmentActionInstall,
				PackageSelection: assignments.PackageSelectionLatestEligible,
				Package: packages.Package{
					ID:                        10,
					SoftwareName:              "GoogleChrome",
					InstallerArtifactID:       &artifactID,
					InstallerArtifactLocation: "apps/GoogleChrome.pkg",
				},
			},
		},
	}
	presigner := serviceArtifactPresigner{url: "https://storage.example/apps/GoogleChrome.pkg?sig=1"}
	service := munki.NewService(nil, store, munki.WithArtifactPresigner(presigner))

	location, err := service.ArtifactRedirect(
		context.Background(),
		munki.ClientHost{ID: 1, Serial: "C02MUNKI"},
		artifacts.ArtifactKindPackage,
		"apps/GoogleChrome.pkg",
	)
	if err != nil {
		t.Fatalf("ArtifactRedirect allowed package: %v", err)
	}
	if location != presigner.url {
		t.Fatalf("location = %q, want presigned URL", location)
	}

	_, err = service.ArtifactRedirect(
		context.Background(),
		munki.ClientHost{ID: 1, Serial: "C02MUNKI"},
		artifacts.ArtifactKindPackage,
		"apps/Blocked.pkg",
	)
	if !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("ArtifactRedirect blocked package error = %v, want ErrNotFound", err)
	}
}

type serviceArtifactStore struct {
	artifacts map[string]artifacts.Artifact
	packages  []assignments.EffectivePackage
}

func (s serviceArtifactStore) GetByLocation(
	_ context.Context,
	kind artifacts.ArtifactKind,
	location string,
) (*artifacts.Artifact, error) {
	artifact, ok := s.artifacts[string(kind)+"/"+location]
	if !ok {
		return nil, dbutil.ErrNotFound
	}
	return &artifact, nil
}

func (s serviceArtifactStore) EffectivePackagesForHost(
	_ context.Context,
	hostID int64,
) ([]assignments.EffectivePackage, error) {
	if hostID != 1 {
		return nil, nil
	}
	return s.packages, nil
}

type serviceArtifactPresigner struct {
	url string
}

func (p serviceArtifactPresigner) PresignGet(_ context.Context, _ artifacts.Artifact) (string, error) {
	return p.url, nil
}

var _ interface {
	GetByLocation(context.Context, artifacts.ArtifactKind, string) (*artifacts.Artifact, error)
	EffectivePackagesForHost(context.Context, int64) ([]assignments.EffectivePackage, error)
} = serviceArtifactStore{}

var _ interface {
	PresignGet(context.Context, artifacts.Artifact) (string, error)
} = serviceArtifactPresigner{}
