package handlers

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

const munkiTag = "Munki"

type munkiListInput struct {
	apitypes.ListQueryInput
}

// MunkiStores groups the Munki admin resource stores used by route registration.
type MunkiStores struct {
	Artifacts      *artifacts.Store
	Packages       *packages.Store
	SoftwareTitles *munkisoftware.Store
}

// RegisterMunki registers admin endpoints for Munki-managed software.
func RegisterMunki(api huma.API, stores MunkiStores, artifactStorage munkiArtifactStorage) {
	registerMunkiSoftware(api, stores.SoftwareTitles, stores.Packages)
	registerMunkiArtifacts(api, stores.Artifacts, artifactStorage)
	registerMunkiPackages(api, stores.Packages)
}

func (input munkiListInput) params() dbutil.ListParams {
	return input.ListQueryInput.Params()
}
