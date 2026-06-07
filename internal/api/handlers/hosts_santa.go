package handlers

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/santa"
)

type santaHostDetailContributor struct {
	loader santaHostStateLoader
}

type santaHostStateLoader interface {
	LoadHostState(context.Context, int64) (*santa.HostState, error)
}

// SantaHostDetailContributor returns a host detail contributor backed by Santa state.
func SantaHostDetailContributor(loader santaHostStateLoader) HostDetailContributor {
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
