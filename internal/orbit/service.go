package orbit

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/models"
)

// Enrollment failures the handler maps to specific HTTP status codes.
var (
	ErrInvalidEnrollSecret = errors.New("invalid enroll secret")
	ErrMissingHardwareUUID = errors.New("hardware_uuid is required")
)

// nodeKeyLength is the Orbit node key length.
const nodeKeyLength = 24

// nodeKeyAlphabet keeps node keys URL-safe.
const nodeKeyAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Service performs Orbit-protocol operations against the host store.
type Service struct {
	hosts          *hosts.HostStore
	secrets        *models.SecretStore
	deviceMappings *hosts.DeviceMappingStore
}

// NewService returns an Orbit service.
func NewService(
	hosts *hosts.HostStore,
	secrets *models.SecretStore,
	deviceMappings *hosts.DeviceMappingStore,
) *Service {
	return &Service{hosts: hosts, secrets: secrets, deviceMappings: deviceMappings}
}

// Enroll validates the request, upserts the host, and returns a fresh node key.
// Re-enrollment of the same hardware UUID overwrites the existing key, so prior
// keys stop authenticating immediately.
func (s *Service) Enroll(ctx context.Context, req EnrollRequest) (*hosts.Host, string, error) {
	if s.hosts == nil || s.secrets == nil {
		return nil, "", errors.New("orbit service is not configured")
	}
	if strings.TrimSpace(req.HardwareUUID) == "" {
		return nil, "", ErrMissingHardwareUUID
	}

	ok, err := s.secrets.ValidateActive(ctx, models.SecretOrbit, req.EnrollSecret)
	if err != nil {
		return nil, "", fmt.Errorf("validate enroll secret: %w", err)
	}
	if !ok {
		return nil, "", ErrInvalidEnrollSecret
	}

	nodeKey, err := generateNodeKey()
	if err != nil {
		return nil, "", fmt.Errorf("generate node key: %w", err)
	}

	host, err := s.hosts.UpsertOnOrbitEnroll(ctx, hosts.EnrollParams{
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
	if _, err := s.hosts.GetByOrbitNodeKey(ctx, nodeKey); err != nil {
		return ConfigResponse{}, err
	}
	return ConfigResponse{Flags: []byte("{}")}, nil
}

// ValidateNodeKey reports whether nodeKey belongs to an active Orbit host.
func (s *Service) ValidateNodeKey(ctx context.Context, nodeKey string) error {
	_, err := s.hosts.GetByOrbitNodeKey(ctx, nodeKey)
	return err
}

// SetDeviceMapping records a profile-provided email for the host.
func (s *Service) SetDeviceMapping(ctx context.Context, nodeKey, email string) error {
	if s.deviceMappings == nil {
		return nil
	}
	host, err := s.hosts.GetByOrbitNodeKey(ctx, nodeKey)
	if err != nil {
		return err
	}
	return s.deviceMappings.Upsert(ctx, host.ID, strings.TrimSpace(email), hosts.DeviceMappingSourceOrbitProfile)
}

func generateNodeKey() (string, error) {
	buf := make([]byte, nodeKeyLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = nodeKeyAlphabet[int(b)%len(nodeKeyAlphabet)]
	}
	return string(buf), nil
}
