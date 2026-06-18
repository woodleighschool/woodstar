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
	affinity, err := s.loadUserAffinity(ctx, []int64{host.ID})
	if err != nil {
		return nil, err
	}

	detailHost := *host
	detailHost.UserAffinity = affinity[host.ID]
	return &HostDetail{
		Host:         detailHost,
		Labels:       hostLabels,
		Users:        hostUsers,
		Batteries:    batteries,
		Certificates: certificates,
	}, nil
}
