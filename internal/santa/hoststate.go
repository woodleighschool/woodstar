package santa

import "context"

type observedHostStateStore interface {
	LoadObservedHostState(context.Context, int64) (*HostState, error)
}

// HostStateService composes Santa host observation with effective configuration.
type HostStateService struct {
	state          observedHostStateStore
	configurations configurationResolver
}

// NewHostStateService returns a Santa host-state loader.
func NewHostStateService(state observedHostStateStore, configurations configurationResolver) *HostStateService {
	return &HostStateService{state: state, configurations: configurations}
}

// LoadHostState returns the Santa detail attached to an existing host, if any.
func (s *HostStateService) LoadHostState(ctx context.Context, hostID int64) (*HostState, error) {
	state, err := s.state.LoadObservedHostState(ctx, hostID)
	if err != nil || state == nil {
		return state, err
	}
	configuration, err := s.configurations.ResolveConfigurationForHost(ctx, hostID)
	if err != nil {
		return nil, err
	}
	if configuration != nil {
		state.EffectiveConfiguration = configuration
	}
	return state, nil
}
