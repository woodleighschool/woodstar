package hosts

import (
	"context"
)

func (s *Store) LoadDetail(ctx context.Context, host *Host) (*HostDetail, error) {
	hostLabels, err := s.labels.ListForHost(ctx, host.ID)
	if err != nil {
		return nil, err
	}
	hostUsers, err := s.ListUsers(ctx, host.ID)
	if err != nil {
		return nil, err
	}
	batteries, err := s.ListBatteries(ctx, host.ID)
	if err != nil {
		return nil, err
	}
	certificates, err := s.ListCertificates(ctx, host.ID)
	if err != nil {
		return nil, err
	}
	primaryUsers, err := s.loadPrimaryUser(ctx, []int64{host.ID})
	if err != nil {
		return nil, err
	}

	detailHost := *host
	primaryUser := primaryUsers[host.ID]
	detailHost.PrimaryUser = primaryUser.Primary
	detailHost.PrimaryUserSources = primaryUser.Sources
	return &HostDetail{
		Host:         detailHost,
		Labels:       hostLabels,
		Users:        hostUsers,
		Batteries:    batteries,
		Certificates: certificates,
	}, nil
}
