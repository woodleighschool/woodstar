package munki_test

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki"
)

func TestServiceArtifactRedirectRequiresEffectivePackage(t *testing.T) {
	artifactID := int64(42)
	store := serviceArtifactStore{
		artifacts: map[string]munki.Artifact{
			"package/apps/GoogleChrome.pkg": {
				ID:         artifactID,
				Kind:       munki.ArtifactKindPackage,
				Location:   "apps/GoogleChrome.pkg",
				StorageKey: "apps/GoogleChrome.pkg",
			},
			"package/apps/Blocked.pkg": {
				ID:         43,
				Kind:       munki.ArtifactKindPackage,
				Location:   "apps/Blocked.pkg",
				StorageKey: "apps/Blocked.pkg",
			},
		},
		packages: []munki.EffectivePackage{
			{
				Package: munki.Package{
					ID:                        10,
					Name:                      "GoogleChrome",
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
		munki.ArtifactKindPackage,
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
		munki.ArtifactKindPackage,
		"apps/Blocked.pkg",
	)
	if !errors.Is(err, munki.ErrNotFound) {
		t.Fatalf("ArtifactRedirect blocked package error = %v, want ErrNotFound", err)
	}
}

type serviceArtifactStore struct {
	artifacts map[string]munki.Artifact
	packages  []munki.EffectivePackage
}

func (s serviceArtifactStore) GetArtifactByLocation(
	_ context.Context,
	kind munki.ArtifactKind,
	location string,
) (*munki.Artifact, error) {
	artifact, ok := s.artifacts[string(kind)+"/"+location]
	if !ok {
		return nil, dbutil.ErrNotFound
	}
	return &artifact, nil
}

func (s serviceArtifactStore) EffectivePackagesForHost(
	_ context.Context,
	hostID int64,
) ([]munki.EffectivePackage, error) {
	if hostID != 1 {
		return nil, nil
	}
	return s.packages, nil
}

type serviceArtifactPresigner struct {
	url string
}

func (p serviceArtifactPresigner) PresignGet(_ context.Context, _ munki.Artifact) (string, error) {
	return p.url, nil
}

var _ interface {
	GetArtifactByLocation(context.Context, munki.ArtifactKind, string) (*munki.Artifact, error)
	EffectivePackagesForHost(context.Context, int64) ([]munki.EffectivePackage, error)
} = serviceArtifactStore{}

var _ interface {
	PresignGet(context.Context, munki.Artifact) (string, error)
} = serviceArtifactPresigner{}
