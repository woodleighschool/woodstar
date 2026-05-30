package orbit

import (
	"context"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// Service performs Orbit-protocol operations against the host store.
type Service struct {
	hostStore         *hosts.Store
	secretStore       *agentauth.Store
	userAffinityStore *hosts.UserAffinityStore
}

func NewService(
	hostStore *hosts.Store,
	secretStore *agentauth.Store,
	userAffinityStore *hosts.UserAffinityStore,
) *Service {
	return &Service{hostStore: hostStore, secretStore: secretStore, userAffinityStore: userAffinityStore}
}

// Enroll validates the request, upserts the host, and returns a fresh node key.
// Re-enrollment of the same hardware UUID overwrites the existing key, so prior
// keys stop authenticating immediately.
func (s *Service) Enroll(ctx context.Context, req EnrollRequest) (*hosts.Host, string, error) {
	if req.HardwareUUID == "" {
		return nil, "", ErrMissingHardwareUUID
	}

	nodeKey, err := IssueNodeKey(ctx, s.secretStore, req.EnrollSecret)
	if err != nil {
		return nil, "", err
	}

	host, err := s.hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware: hosts.HostHardware{
			UUID:            req.HardwareUUID,
			Serial:          req.HardwareSerial,
			ModelIdentifier: req.HardwareModel,
		},
		Hostname:     req.Hostname,
		ComputerName: req.ComputerName,
		OrbitNodeKey: nodeKey,
	})
	if err != nil {
		return nil, "", fmt.Errorf("upsert host: %w", err)
	}
	return host, nodeKey, nil
}

// Config returns the current Orbit config.
func (s *Service) Config(ctx context.Context, nodeKey string) (ConfigResponse, error) {
	if _, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey); err != nil {
		return ConfigResponse{}, err
	}
	return ConfigResponse{}, nil
}

// ValidateNodeKey reports whether nodeKey belongs to an active Orbit host.
func (s *Service) ValidateNodeKey(ctx context.Context, nodeKey string) error {
	_, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	return err
}

// SetUserAffinity records a profile-provided email for the host.
func (s *Service) SetUserAffinity(ctx context.Context, nodeKey, email string) error {
	host, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	if err != nil {
		return err
	}
	return s.userAffinityStore.Upsert(ctx, host.ID, email, hosts.UserAffinitySourceOrbitProfile)
}
