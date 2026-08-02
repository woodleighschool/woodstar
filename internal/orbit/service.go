package orbit

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

const orbitCommandLineStartupFlags = `{
	"disable_carver": true,
	"carver_disable_function": true,
	"logger_min_status": 4
}`

// ErrInvalidDeviceAuthToken reports a token outside Orbit's canonical UUID form.
var ErrInvalidDeviceAuthToken = errors.New("invalid Orbit device auth token")

// EnrollmentService performs Orbit enrollment and config operations.
type EnrollmentService struct {
	hostStore    hostStore
	secretStore  agentauth.SecretVerifier
	primaryUsers primaryUserStore
	heartbeats   heartbeatRecorder
}

type hostStore interface {
	UpsertOnOrbitEnroll(context.Context, hosts.InventoryUpdate) (*hosts.Host, error)
	GetByOrbitNodeKey(context.Context, string) (*hosts.Host, error)
	SetOrbitDeviceAuthToken(context.Context, string, string) error
	ValidateOrbitDeviceAuthToken(context.Context, string) (*hosts.Host, error)
}

type primaryUserStore interface {
	Upsert(context.Context, int64, string, hosts.PrimaryUserSource) error
}

type heartbeatRecorder interface {
	Record(context.Context, int64, heartbeats.Source, heartbeats.Contact) error
}

func NewEnrollmentService(
	hostStore hostStore,
	secretStore agentauth.SecretVerifier,
	primaryUsers primaryUserStore,
	heartbeats heartbeatRecorder,
) *EnrollmentService {
	return &EnrollmentService{
		hostStore: hostStore, secretStore: secretStore, primaryUsers: primaryUsers, heartbeats: heartbeats,
	}
}

// Enroll validates the request, upserts the host, and returns a fresh node key.
// Re-enrollment of the same hardware UUID overwrites the existing key, so prior
// keys stop authenticating immediately.
func (s *EnrollmentService) Enroll(ctx context.Context, req EnrollRequest, contact heartbeats.Contact) (*hosts.Host, string, error) {
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
	if err := s.heartbeats.Record(ctx, host.ID, heartbeats.SourceOrbit, contact); err != nil {
		return nil, "", fmt.Errorf("record heartbeat: %w", err)
	}
	return host, nodeKey, nil
}

// Config returns the current Orbit config.
func (s *EnrollmentService) Config(ctx context.Context, nodeKey string, contact heartbeats.Contact) (ConfigResponse, error) {
	host, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	if err != nil {
		return ConfigResponse{}, err
	}
	if err := s.heartbeats.Record(ctx, host.ID, heartbeats.SourceOrbit, contact); err != nil {
		return ConfigResponse{}, fmt.Errorf("record heartbeat: %w", err)
	}
	return ConfigResponse{CommandLineStartupFlags: json.RawMessage(orbitCommandLineStartupFlags)}, nil
}

// SetPrimaryUser records a profile-provided email for the host.
func (s *EnrollmentService) SetPrimaryUser(ctx context.Context, nodeKey, email string, contact heartbeats.Contact) error {
	host, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	if err != nil {
		return err
	}
	if err := s.heartbeats.Record(ctx, host.ID, heartbeats.SourceOrbit, contact); err != nil {
		return fmt.Errorf("record heartbeat: %w", err)
	}
	return s.primaryUsers.Upsert(ctx, host.ID, email, hosts.PrimaryUserSourceOrbitProfile)
}

// SetDeviceAuthToken rotates the per-host token issued and retained by Orbit.
func (s *EnrollmentService) SetDeviceAuthToken(ctx context.Context, nodeKey, token string, contact heartbeats.Contact) error {
	if err := validateDeviceAuthToken(token); err != nil {
		return err
	}
	host, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	if err != nil {
		return err
	}
	if err := s.heartbeats.Record(ctx, host.ID, heartbeats.SourceOrbit, contact); err != nil {
		return fmt.Errorf("record heartbeat: %w", err)
	}
	return s.hostStore.SetOrbitDeviceAuthToken(ctx, nodeKey, token)
}

func validateDeviceAuthToken(token string) error {
	compact := strings.ReplaceAll(token, "-", "")
	_, err := hex.DecodeString(compact)
	if err != nil || len(token) != 36 || len(compact) != 32 ||
		token[8] != '-' || token[13] != '-' || token[18] != '-' || token[23] != '-' ||
		token != strings.ToLower(token) {
		return ErrInvalidDeviceAuthToken
	}
	return nil
}

// ValidateDeviceAuthToken checks whether an Orbit machine token is active.
func (s *EnrollmentService) ValidateDeviceAuthToken(ctx context.Context, token string, contact heartbeats.Contact) error {
	host, err := s.hostStore.ValidateOrbitDeviceAuthToken(ctx, token)
	if err != nil {
		return err
	}
	if err := s.heartbeats.Record(ctx, host.ID, heartbeats.SourceOrbit, contact); err != nil {
		return fmt.Errorf("record heartbeat: %w", err)
	}
	return nil
}
