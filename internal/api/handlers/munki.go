package handlers

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
	"github.com/woodleighschool/woodstar/internal/munki/assignments"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/softwaretitles"
)

const munkiTag = "Munki"

type munkiListInput struct {
	ListQueryInput
}

// MunkiStores groups the Munki admin resource stores used by route registration.
type MunkiStores struct {
	Artifacts      *artifacts.Store
	Assignments    *assignments.Store
	Packages       *packages.Store
	SoftwareTitles *softwaretitles.Store
}

// RegisterMunki registers admin endpoints for Munki-managed software.
func RegisterMunki(api huma.API, stores MunkiStores, artifactStorage munkiArtifactStorage) {
	registerMunkiSoftwareTitles(api, stores.SoftwareTitles, stores.Packages, stores.Assignments)
	registerMunkiArtifacts(api, stores.Artifacts, artifactStorage)
	registerMunkiPackages(api, stores.Packages)
}

func (input munkiListInput) params() dbutil.ListParams {
	return input.ListQueryInput.params()
}
