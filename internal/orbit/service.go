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
	hostStore    *hosts.Store
	secretStore  *agentauth.Store
	primaryUsers *hosts.PrimaryUserStore
}

func NewEnrollmentService(
	hostStore *hosts.Store,
	secretStore *agentauth.Store,
	primaryUsers *hosts.PrimaryUserStore,
) *EnrollmentService {
	return &EnrollmentService{hostStore: hostStore, secretStore: secretStore, primaryUsers: primaryUsers}
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
	return ConfigResponse{CommandLineStartupFlags: json.RawMessage(orbitCommandLineStartupFlags)}, nil
}

// SetPrimaryUser records a profile-provided email for the host.
func (s *EnrollmentService) SetPrimaryUser(ctx context.Context, nodeKey, email string) error {
	host, err := s.hostStore.GetByOrbitNodeKey(ctx, nodeKey)
	if err != nil {
		return err
	}
	return s.primaryUsers.Upsert(ctx, host.ID, email, hosts.PrimaryUserSourceOrbitProfile)
}

// SetDeviceAuthToken rotates the per-host token issued and retained by Orbit.
func (s *EnrollmentService) SetDeviceAuthToken(ctx context.Context, nodeKey, token string) error {
	compact := strings.ReplaceAll(token, "-", "")
	_, err := hex.DecodeString(compact)
	if err != nil || len(token) != 36 || len(compact) != 32 ||
		token[8] != '-' || token[13] != '-' || token[18] != '-' || token[23] != '-' ||
		token != strings.ToLower(token) {
		return ErrInvalidDeviceAuthToken
	}
	return s.hostStore.SetOrbitDeviceAuthToken(ctx, nodeKey, token)
}

// ValidateDeviceAuthToken checks whether an Orbit machine token is active.
func (s *EnrollmentService) ValidateDeviceAuthToken(ctx context.Context, token string) error {
	return s.hostStore.ValidateOrbitDeviceAuthToken(ctx, token)
}
