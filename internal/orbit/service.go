package orbit

import (
	"context"
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// Service performs Orbit-protocol operations against the host store.
type Service struct {
	hostStore          *hosts.Store
	secretStore        *agentauth.Store
	deviceMappingStore *hosts.DeviceMappingStore
}

func NewService(
	hostStore *hosts.Store,
	secretStore *agentauth.Store,
	deviceMappingStore *hosts.DeviceMappingStore,
) *Service {
	return &Service{hostStore: hostStore, secretStore: secretStore, deviceMappingStore: deviceMappingStore}
}

// Enroll validates the request, upserts the host, and returns a fresh node key.
// Re-enrollment of the same hardware UUID overwrites the existing key, so prior
// keys stop authenticating immediately.
func (s *Service) Enroll(ctx context.Context, req EnrollRequest) (*hosts.Host, string, error) {
	if strings.TrimSpace(req.HardwareUUID) == "" {
		return nil, "", ErrMissingHardwareUUID
	}

	nodeKey, err := IssueNodeKey(ctx, s.secretStore, req.EnrollSecret)
	if err != nil {
		return nil, "", err
	}

	host, err := s.hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID:   req.HardwareUUID,
		HardwareSerial: req.HardwareSerial,
		Hostname:       req.Hostname,
		ComputerName:   req.ComputerName,
		HardwareModel:  req.HardwareModel,
		Platform:       req.Platform,
		PlatformLike:   req.PlatformLike,
		OrbitNodeKey:   nodeKey,
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

// SetDeviceMapping records a profile-provided email for the host.
func (s *Service) SetDeviceMapping(ctx context.Context, nodeKey, email string) error {
	host, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	if err != nil {
		return err
	}
	return s.deviceMappingStore.Upsert(ctx, host.ID, strings.TrimSpace(email), hosts.DeviceMappingSourceOrbitProfile)
}
