package orbit

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
)

const orbitCommandLineStartupFlags = `{
	"disable_carver": true,
	"carver_disable_function": true,
	"logger_plugin": "tls",
	"logger_min_status": 2
}`

// ErrInvalidDeviceAuthToken reports a token outside Orbit's canonical UUID form.
var ErrInvalidDeviceAuthToken = errors.New("invalid Orbit device auth token")

// EnrollmentService performs Orbit enrollment and config operations.
type EnrollmentService struct {
	hostStore              hostStore
	secretStore            agentauth.SecretVerifier
	primaryUsers           primaryUserStore
	heartbeats             heartbeatRecorder
	remediations           remediationStore
	scriptExecutionTimeout time.Duration
	activity               activity.Recorder
	logger                 *slog.Logger
}

// Dependencies are Orbit's enrollment and runtime collaborators.
type Dependencies struct {
	Hosts                  hostStore
	Secrets                agentauth.SecretVerifier
	PrimaryUsers           primaryUserStore
	Heartbeats             heartbeatRecorder
	Remediations           remediationStore
	ScriptExecutionTimeout time.Duration
	Activity               activity.Recorder
	Logger                 *slog.Logger
}

type hostStore interface {
	UpsertOnOrbitEnroll(
		context.Context,
		hosts.InventoryUpdate,
		heartbeats.Contact,
	) (*hosts.Host, error)
	IDByOrbitNodeKey(context.Context, string) (int64, error)
	SetOrbitDeviceAuthToken(context.Context, string, string) error
	IDByOrbitDeviceAuthToken(context.Context, string) (int64, error)
}

type primaryUserStore interface {
	Upsert(context.Context, int64, string, hosts.PrimaryUserSource) error
}

type heartbeatRecorder interface {
	Record(context.Context, int64, heartbeats.Source, heartbeats.Contact) error
}

type remediationStore interface {
	PendingRemediationExecutionIDs(context.Context, int64) ([]string, error)
	RemediationExecution(context.Context, int64, string) (*policies.RemediationExecution, error)
	RecordRemediationResult(context.Context, int64, policies.RemediationResult) error
}

func NewEnrollmentService(deps Dependencies) *EnrollmentService {
	return &EnrollmentService{
		hostStore: deps.Hosts, secretStore: deps.Secrets, primaryUsers: deps.PrimaryUsers,
		heartbeats: deps.Heartbeats, remediations: deps.Remediations,
		scriptExecutionTimeout: deps.ScriptExecutionTimeout,
		activity:               deps.Activity, logger: deps.Logger,
	}
}

// Enroll validates the request, upserts the host, and returns a fresh node key.
// Re-enrollment reuses the existing host identity and replaces its Orbit node key.
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
	}, contact)
	if err != nil {
		return nil, "", fmt.Errorf("upsert host: %w", err)
	}
	activity.RecordSystem(
		ctx,
		s.activity,
		s.logger,
		activity.AreaOsquery,
		activity.ActionOrbitHostEnrolled,
		activity.Resource("host", host.ID, host.DisplayName),
	)
	return host, nodeKey, nil
}

// Config returns the current Orbit config.
func (s *EnrollmentService) Config(ctx context.Context, nodeKey string, contact heartbeats.Contact) (ConfigResponse, error) {
	hostID, err := s.hostByNodeKey(ctx, nodeKey, contact)
	if err != nil {
		return ConfigResponse{}, err
	}
	pending, err := s.remediations.PendingRemediationExecutionIDs(ctx, hostID)
	if err != nil {
		return ConfigResponse{}, fmt.Errorf("list pending remediations: %w", err)
	}
	return ConfigResponse{
		CommandLineStartupFlags: json.RawMessage(orbitCommandLineStartupFlags),
		ScriptExecutionTimeout:  int(s.scriptExecutionTimeout / time.Second),
		Notifications: Notifications{
			PendingScriptExecutionIDs: pending,
		},
	}, nil
}

// GetScript authenticates Orbit and returns an advertised script without consuming it.
func (s *EnrollmentService) GetScript(
	ctx context.Context,
	req ScriptRequest,
	contact heartbeats.Contact,
) (*ScriptResponse, error) {
	hostID, err := s.hostByNodeKey(ctx, req.OrbitNodeKey, contact)
	if err != nil {
		return nil, err
	}
	execution, err := s.remediations.RemediationExecution(ctx, hostID, req.ExecutionID)
	if err != nil {
		return nil, err
	}
	runtime := 0
	if execution.RuntimeSeconds != nil {
		runtime = *execution.RuntimeSeconds
	}
	return &ScriptResponse{
		HostID:         execution.HostID,
		ExecutionID:    execution.ExecutionID,
		ScriptContents: execution.ScriptContents,
		Output:         execution.Output,
		Runtime:        runtime,
		ExitCode:       execution.ExitCode,
	}, nil
}

// RecordScriptResult authenticates Orbit and records its first terminal report.
func (s *EnrollmentService) RecordScriptResult(
	ctx context.Context,
	req ScriptResult,
	contact heartbeats.Contact,
) error {
	hostID, err := s.hostByNodeKey(ctx, req.OrbitNodeKey, contact)
	if err != nil {
		return err
	}
	return s.remediations.RecordRemediationResult(ctx, hostID, policies.RemediationResult{
		ExecutionID:    req.ExecutionID,
		Output:         req.Output,
		RuntimeSeconds: req.Runtime,
		ExitCode:       req.ExitCode,
	})
}

// SetPrimaryUser records a profile-provided email for the host.
func (s *EnrollmentService) SetPrimaryUser(ctx context.Context, nodeKey, email string, contact heartbeats.Contact) error {
	hostID, err := s.hostByNodeKey(ctx, nodeKey, contact)
	if err != nil {
		return err
	}
	return s.primaryUsers.Upsert(ctx, hostID, email, hosts.PrimaryUserSourceOrbitProfile)
}

// SetDeviceAuthToken rotates the per-host token issued and retained by Orbit.
func (s *EnrollmentService) SetDeviceAuthToken(ctx context.Context, nodeKey, token string, contact heartbeats.Contact) error {
	if err := validateDeviceAuthToken(token); err != nil {
		return err
	}
	if _, err := s.hostByNodeKey(ctx, nodeKey, contact); err != nil {
		return err
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
	hostID, err := s.hostStore.IDByOrbitDeviceAuthToken(ctx, token)
	if err != nil {
		return err
	}
	if err := s.heartbeats.Record(ctx, hostID, heartbeats.SourceOrbit, contact); err != nil {
		return fmt.Errorf("record heartbeat: %w", err)
	}
	return nil
}

func (s *EnrollmentService) hostByNodeKey(ctx context.Context, key string, contact heartbeats.Contact) (int64, error) {
	id, err := s.hostStore.IDByOrbitNodeKey(ctx, key)
	if err != nil {
		return 0, err
	}
	if err := s.heartbeats.Record(ctx, id, heartbeats.SourceOrbit, contact); err != nil {
		return 0, fmt.Errorf("record heartbeat: %w", err)
	}
	return id, nil
}
