// Package santa coordinates host state, events, rules, and sync configuration.
package santa

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

type observedHostStateStore interface {
	LoadObservedHostState(ctx context.Context, hostID int64) (*HostState, error)
}

type configurationWithTargetsResolver interface {
	ResolveConfigurationForHostWithTargets(
		ctx context.Context,
		hostID int64,
	) (*configurations.ConfigurationMatch, error)
}

// HostStateService composes Santa host observation with the matching configuration.
type HostStateService struct {
	state          observedHostStateStore
	configurations configurationWithTargetsResolver
}

// NewHostStateService returns a Santa host-state loader.
func NewHostStateService(
	state observedHostStateStore,
	configurations configurationWithTargetsResolver,
) *HostStateService {
	return &HostStateService{state: state, configurations: configurations}
}

// LoadHostState returns the Santa detail attached to an existing host, if any.
func (s *HostStateService) LoadHostState(ctx context.Context, hostID int64) (*HostState, error) {
	state, err := s.state.LoadObservedHostState(ctx, hostID)
	if err != nil || state == nil {
		return state, err
	}
	configuration, err := s.configurations.ResolveConfigurationForHostWithTargets(ctx, hostID)
	if err != nil {
		return nil, err
	}
	if configuration != nil {
		state.Configuration = configuration
	}
	return state, nil
}
