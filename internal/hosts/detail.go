package hosts

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/labels"
)

func (s *Store) LoadDetail(ctx context.Context, host *Host) (*HostDetail, error) {
	hostLabels, err := labels.NewStore(s.db).ListForHost(ctx, host.ID)
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
	mappings, err := s.q.ListHostDeviceMappings(ctx, sqlc.ListHostDeviceMappingsParams{HostID: host.ID})
	if err != nil {
		return nil, err
	}

	detailHost := *host
	detailHost.DeviceMappings = groupHostDeviceMappings(mappings, 1)[host.ID]
	return &HostDetail{
		Host:         detailHost,
		Labels:       hostLabels,
		Users:        hostUsers,
		Batteries:    batteries,
		Certificates: certificates,
	}, nil
}
