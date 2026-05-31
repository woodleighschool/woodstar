package handlers

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/munki"
)

type munkiHostDetailContributor struct {
	loader munkiHostStateLoader
}

type munkiHostStateLoader interface {
	LoadHostState(context.Context, int64) (*munki.HostMunkiState, error)
}

// MunkiHostDetailContributor returns a host detail contributor backed by Munki state.
func MunkiHostDetailContributor(loader munkiHostStateLoader) HostDetailContributor {
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
