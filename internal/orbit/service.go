package orbit

import (
	"context"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// EnrollmentService performs Orbit enrollment and config operations.
type EnrollmentService struct {
	hostStore        *hosts.Store
	secretStore      *agentauth.Store
	primaryUserStore *hosts.PrimaryUserStore
}

func NewEnrollmentService(
	hostStore *hosts.Store,
	secretStore *agentauth.Store,
	primaryUserStore *hosts.PrimaryUserStore,
) *EnrollmentService {
	return &EnrollmentService{hostStore: hostStore, secretStore: secretStore, primaryUserStore: primaryUserStore}
}

// Enroll validates the request, upserts the host, and returns a fresh node key.
// Re-enrollment of the same hardware UUID overwrites the existing key, so prior
// keys stop authenticating immediately.
func (s *EnrollmentService) Enroll(ctx context.Context, req EnrollRequest) (*hosts.Host, string, error) {
	if req.HardwareUUID == "" {
		return nil, "", enrollment.ErrMissingHardwareUUID
	}

	nodeKey, err := enrollment.IssueNodeKey(ctx, s.secretStore, req.EnrollSecret)
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
func (s *EnrollmentService) Config(ctx context.Context, nodeKey string) (ConfigResponse, error) {
	if _, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey); err != nil {
		return ConfigResponse{}, err
	}
	return ConfigResponse{}, nil
}

// ValidateNodeKey reports whether nodeKey belongs to an active Orbit host.
func (s *EnrollmentService) ValidateNodeKey(ctx context.Context, nodeKey string) error {
	_, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	return err
}

// SetPrimaryUser records a profile-provided email for the host.
func (s *EnrollmentService) SetPrimaryUser(ctx context.Context, nodeKey, email string) error {
	host, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	if err != nil {
		return err
	}
	return s.primaryUserStore.Upsert(ctx, host.ID, email, hosts.PrimaryUserSourceOrbitProfile)
}
