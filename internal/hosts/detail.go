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
	mappings, err := s.q.ListHostUserAffinityMappings(ctx, sqlc.ListHostUserAffinityMappingsParams{HostID: host.ID})
	if err != nil {
		return nil, err
	}

	detailHost := *host
	detailHost.UserAffinity.Mappings = groupHostUserAffinityMappings(mappings, 1)[host.ID]
	if detailHost.UserAffinity.Mappings == nil {
		detailHost.UserAffinity.Mappings = []HostUserAffinityMapping{}
	}
	detailHost.UserAffinity.Primary, err = s.loadHostUserAffinityPrimary(ctx, host.ID)
	if err != nil {
		return nil, err
	}
	return &HostDetail{
		Host:         detailHost,
		Labels:       hostLabels,
		Users:        hostUsers,
		Batteries:    batteries,
		Certificates: certificates,
	}, nil
}
