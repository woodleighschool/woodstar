package handlers

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki"
)

const munkiTag = "Munki"

type munkiListInput struct {
	ListQueryInput
}

// RegisterMunki registers admin endpoints for Munki-managed software.
func RegisterMunki(api huma.API, state *munki.Store, artifactStorage munkiArtifactStorage) {
	registerMunkiSoftwareTitles(api, state)
	registerMunkiArtifacts(api, state, artifactStorage)
	registerMunkiPackages(api, state)
	registerMunkiAssignments(api, state)
}

func (input munkiListInput) params() dbutil.ListParams {
	return input.ListQueryInput.params()
}
