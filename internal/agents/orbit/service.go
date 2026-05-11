package orbit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/agents"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/secrets"
)

const emptyConfigFlags = "{}"

// Service performs Orbit-protocol operations against the host store.
type Service struct {
	hostStore          *hosts.HostStore
	secretStore        *secrets.Store
	deviceMappingStore *hosts.DeviceMappingStore
}

// NewService returns an Orbit service.
func NewService(
	hostStore *hosts.HostStore,
	secrets *secrets.Store,
	deviceMappings *hosts.DeviceMappingStore,
) *Service {
	return &Service{hostStore: hostStore, secretStore: secrets, deviceMappingStore: deviceMappings}
}

// Enroll validates the request, upserts the host, and returns a fresh node key.
// Re-enrollment of the same hardware UUID overwrites the existing key, so prior
// keys stop authenticating immediately.
func (s *Service) Enroll(ctx context.Context, req EnrollRequest) (*hosts.Host, string, error) {
	if strings.TrimSpace(req.HardwareUUID) == "" {
		return nil, "", agents.ErrMissingHardwareUUID
	}

	ok, err := s.secretStore.HasActiveOrbitEnrollSecret(ctx, req.EnrollSecret)
	if err != nil {
		return nil, "", fmt.Errorf("validate enroll secret: %w", err)
	}
	if !ok {
		return nil, "", agents.ErrInvalidEnrollSecret
	}

	nodeKey, err := agents.GenerateNodeKey()
	if err != nil {
		return nil, "", fmt.Errorf("generate node key: %w", err)
	}

	host, err := s.hostStore.UpsertOnOrbitEnroll(ctx, hosts.EnrollParams{
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
	return ConfigResponse{Flags: json.RawMessage(emptyConfigFlags)}, nil
}

// ValidateNodeKey reports whether nodeKey belongs to an active Orbit host.
func (s *Service) ValidateNodeKey(ctx context.Context, nodeKey string) error {
	_, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	return err
}

// SetDeviceMapping records a profile-provided email for the host.
func (s *Service) SetDeviceMapping(ctx context.Context, nodeKey, email string) error {
	if s.deviceMappingStore == nil {
		return nil
	}
	host, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	if err != nil {
		return err
	}
	return s.deviceMappingStore.Upsert(ctx, host.ID, strings.TrimSpace(email), hosts.DeviceMappingSourceOrbitProfile)
}
