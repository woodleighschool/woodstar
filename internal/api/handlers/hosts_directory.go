package handlers

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/hosts"
)

type directoryHostDetailContributor struct {
	loader hostUserAffinityLoader
}

type hostUserAffinityLoader interface {
	LoadHostUserAffinity(context.Context, int64) (*hosts.HostUserAffinity, error)
}

// DirectoryHostDetailContributor returns a host detail contributor backed by
// directory-owned user affinity enrichment.
func DirectoryHostDetailContributor(loader hostUserAffinityLoader) HostDetailContributor {
	if loader == nil {
		return nil
	}
	return directoryHostDetailContributor{loader: loader}
}

func (c directoryHostDetailContributor) ContributeHostDetail(
	ctx context.Context,
	hostID int64,
	body *hostDetailBody,
) error {
	affinity, err := c.loader.LoadHostUserAffinity(ctx, hostID)
	if err != nil {
		return err
	}
	body.UserAffinity = affinity
	return nil
}
