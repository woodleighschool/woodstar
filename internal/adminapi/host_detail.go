package adminapi

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/santa"
)

// HostDetail is the admin host detail response with optional capability fields.
type HostDetail struct {
	hosts.HostDetail

	Munki *munki.HostState `json:"munki,omitempty"`
	Santa *santa.HostState `json:"santa,omitempty"`
}

// HostRoutesOptions assembles the host admin route options: the osquery check
// status filter plus the munki and santa host-detail contributors. It lives
// here, next to the adapters it builds, so the wiring layer hands it plain
// stores rather than reaching into capability internals.
func HostRoutesOptions(
	store *hosts.Store,
	userAffinities *hosts.UserAffinityStore,
	checkStore *checks.Store,
	munkiState munkiHostStateLoader,
	santaState santaHostStateLoader,
) hosts.AdminRoutesOptions[HostDetail] {
	var checkFilter hosts.CheckStatusFilter
	if checkStore != nil {
		checkFilter = osqueryCheckFilter{store: checkStore}
	}
	return hosts.AdminRoutesOptions[HostDetail]{
		Store:          store,
		UserAffinities: userAffinities,
		CheckFilter:    checkFilter,
		DetailBuilder:  func(detail hosts.HostDetail) HostDetail { return HostDetail{HostDetail: detail} },
		Contributors: []hosts.DetailContributor[HostDetail]{
			newMunkiHostDetailContributor(munkiState),
			newSantaHostDetailContributor(santaState),
		},
	}
}

type osqueryCheckFilter struct {
	store *checks.Store
}

func (f osqueryCheckFilter) HostIDsByStatus(ctx context.Context, checkID int64, status string) ([]int64, error) {
	return f.store.HostIDsByStatus(ctx, checkID, checks.CheckStatus(status))
}

type munkiHostDetailContributor struct {
	loader munkiHostStateLoader
}

type munkiHostStateLoader interface {
	LoadHostState(context.Context, int64) (*munki.HostState, error)
}

func newMunkiHostDetailContributor(loader munkiHostStateLoader) hosts.DetailContributor[HostDetail] {
	if loader == nil {
		return nil
	}
	return munkiHostDetailContributor{loader: loader}
}

func (c munkiHostDetailContributor) ContributeHostDetail(
	ctx context.Context,
	hostID int64,
	body *HostDetail,
) error {
	detail, err := c.loader.LoadHostState(ctx, hostID)
	if err != nil {
		return err
	}
	body.Munki = detail
	return nil
}

type santaHostDetailContributor struct {
	loader santaHostStateLoader
}

type santaHostStateLoader interface {
	LoadHostState(context.Context, int64) (*santa.HostState, error)
}

func newSantaHostDetailContributor(loader santaHostStateLoader) hosts.DetailContributor[HostDetail] {
	if loader == nil {
		return nil
	}
	return santaHostDetailContributor{loader: loader}
}

func (c santaHostDetailContributor) ContributeHostDetail(
	ctx context.Context,
	hostID int64,
	body *HostDetail,
) error {
	detail, err := c.loader.LoadHostState(ctx, hostID)
	if err != nil {
		return err
	}
	body.Santa = detail
	return nil
}
