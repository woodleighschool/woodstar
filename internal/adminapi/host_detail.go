package adminapi

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/adminapi/adminctx"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki/hoststate"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/santa"
)

func requireAdminUser(ctx context.Context) error {
	_, err := adminctx.RequireAdmin(ctx)
	return err
}

// HostDetail is the admin host detail response with optional capability fields.
type HostDetail struct {
	hosts.HostDetail
	Munki *hoststate.State `json:"munki,omitempty"`
	Santa *santa.HostState `json:"santa,omitempty"`
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
	LoadHostState(context.Context, int64) (*hoststate.State, error)
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
