package orbit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
)

func TestConfigResponseWireShapeMatchesOrbit(t *testing.T) {
	body, err := json.Marshal(ConfigResponse{
		CommandLineStartupFlags: json.RawMessage(orbitCommandLineStartupFlags),
	})
	if err != nil {
		t.Fatalf("marshal config response: %v", err)
	}

	var got struct {
		CommandLineStartupFlags map[string]any `json:"command_line_startup_flags"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal config response: %v", err)
	}
	flags := got.CommandLineStartupFlags
	if flags["disable_carver"] != true ||
		flags["carver_disable_function"] != true ||
		flags["logger_plugin"] != "tls" ||
		flags["logger_min_status"] != float64(2) {
		t.Fatalf("command-line flags = %#v", flags)
	}
}

func TestValidateDeviceAuthToken(t *testing.T) {
	t.Parallel()

	const valid = "11111111-2222-4333-8444-555555555555"
	if err := validateDeviceAuthToken(valid); err != nil {
		t.Fatalf("validate canonical token: %v", err)
	}

	for name, token := range map[string]string{
		"blank":           "",
		"not hexadecimal": "zzzzzzzz-2222-4333-8444-555555555555",
		"missing hyphens": "11111111222243338444555555555555",
		"uppercase":       "AAAAAAAA-BBBB-4CCC-8DDD-EEEEEEEEEEEE",
		"too long":        valid + "0",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := validateDeviceAuthToken(token); !errors.Is(err, ErrInvalidDeviceAuthToken) {
				t.Fatalf("validate token error = %v, want ErrInvalidDeviceAuthToken", err)
			}
		})
	}
}

func TestEnrollmentServiceRecordsAuthenticatedHostContact(t *testing.T) {
	t.Parallel()

	contact := heartbeats.Contact{RemoteIP: "203.0.113.42", UserAgent: "Orbit/1.2.3"}
	tests := []struct {
		name string
		call func(t *testing.T, service *EnrollmentService)
	}{
		{
			name: "enroll",
			call: func(t *testing.T, service *EnrollmentService) {
				t.Helper()
				if _, _, err := service.Enroll(t.Context(), EnrollRequest{EnrollSecret: "secret", HardwareUUID: "hardware-uuid"}, contact); err != nil {
					t.Fatalf("Enroll: %v", err)
				}
			},
		},
		{
			name: "config",
			call: func(t *testing.T, service *EnrollmentService) {
				t.Helper()
				if _, err := service.Config(t.Context(), "node-key", contact); err != nil {
					t.Fatalf("Config: %v", err)
				}
			},
		},
		{
			name: "device mapping",
			call: func(t *testing.T, service *EnrollmentService) {
				t.Helper()
				if err := service.SetPrimaryUser(t.Context(), "node-key", "person@example.test", contact); err != nil {
					t.Fatalf("SetPrimaryUser: %v", err)
				}
			},
		},
		{
			name: "device token rotation",
			call: func(t *testing.T, service *EnrollmentService) {
				t.Helper()
				if err := service.SetDeviceAuthToken(t.Context(), "node-key", "11111111-2222-4333-8444-555555555555", contact); err != nil {
					t.Fatalf("SetDeviceAuthToken: %v", err)
				}
			},
		},
		{
			name: "device ping",
			call: func(t *testing.T, service *EnrollmentService) {
				t.Helper()
				if err := service.ValidateDeviceAuthToken(t.Context(), "device-token", contact); err != nil {
					t.Fatalf("ValidateDeviceAuthToken: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			recorder := &fakeHeartbeatRecorder{}
			hostStore := &fakeOrbitHostStore{host: &hosts.Host{ID: 42}}
			service := newTestEnrollmentService(recorder, hostStore)

			tt.call(t, service)

			if len(recorder.calls) != 1 {
				t.Fatalf("Record calls = %d, want 1", len(recorder.calls))
			}
			got := recorder.calls[0]
			if got.hostID != 42 || got.source != heartbeats.SourceOrbit || got.contact != contact {
				t.Fatalf("Record call = %#v, want host 42, orbit, %#v", got, contact)
			}
		})
	}
}

func TestEnrollmentServiceDoesNotRecordUnauthenticatedContact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(t *testing.T, service *EnrollmentService, hostStore *fakeOrbitHostStore)
	}{
		{
			name: "invalid enroll secret",
			call: func(t *testing.T, service *EnrollmentService, _ *fakeOrbitHostStore) {
				t.Helper()
				_, _, err := service.Enroll(t.Context(), EnrollRequest{EnrollSecret: "bad", HardwareUUID: "hardware-uuid"}, heartbeats.Contact{})
				if !errors.Is(err, agentauth.ErrInvalidSecret) {
					t.Fatalf("Enroll error = %v, want invalid secret", err)
				}
			},
		},
		{
			name: "invalid node key",
			call: func(t *testing.T, service *EnrollmentService, hostStore *fakeOrbitHostStore) {
				t.Helper()
				hostStore.getErr = fault.ErrNotFound
				_, err := service.Config(t.Context(), "bad", heartbeats.Contact{})
				if !errors.Is(err, fault.ErrNotFound) {
					t.Fatalf("Config error = %v, want not found", err)
				}
			},
		},
		{
			name: "invalid device token",
			call: func(t *testing.T, service *EnrollmentService, _ *fakeOrbitHostStore) {
				t.Helper()
				err := service.SetDeviceAuthToken(t.Context(), "node-key", "invalid", heartbeats.Contact{})
				if !errors.Is(err, ErrInvalidDeviceAuthToken) {
					t.Fatalf("SetDeviceAuthToken error = %v, want invalid token", err)
				}
			},
		},
		{
			name: "unknown device token",
			call: func(t *testing.T, service *EnrollmentService, hostStore *fakeOrbitHostStore) {
				t.Helper()
				hostStore.validateErr = fault.ErrNotFound
				err := service.ValidateDeviceAuthToken(t.Context(), "bad", heartbeats.Contact{})
				if !errors.Is(err, fault.ErrNotFound) {
					t.Fatalf("ValidateDeviceAuthToken error = %v, want not found", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			recorder := &fakeHeartbeatRecorder{}
			hostStore := &fakeOrbitHostStore{host: &hosts.Host{ID: 42}}
			service := newTestEnrollmentService(recorder, hostStore)
			if tt.name == "invalid enroll secret" {
				service.secretStore = fakeOrbitSecretVerifier{ok: false}
			}

			tt.call(t, service, hostStore)
			if len(recorder.calls) != 0 {
				t.Fatalf("Record calls = %d, want 0", len(recorder.calls))
			}
		})
	}
}

func TestEnrollmentServicePropagatesHeartbeatError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("heartbeat unavailable")
	hostStore := &fakeOrbitHostStore{host: &hosts.Host{ID: 42}}
	service := newTestEnrollmentService(&fakeHeartbeatRecorder{err: wantErr}, hostStore)
	_, err := service.Config(t.Context(), "node-key", heartbeats.Contact{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Config error = %v, want %v", err, wantErr)
	}
}

type fakeOrbitHostStore struct {
	host        *hosts.Host
	getErr      error
	validateErr error
}

func (s *fakeOrbitHostStore) UpsertOnOrbitEnroll(context.Context, hosts.InventoryUpdate) (*hosts.Host, error) {
	return s.host, nil
}

func (s *fakeOrbitHostStore) GetByOrbitNodeKey(context.Context, string) (*hosts.Host, error) {
	return s.host, s.getErr
}

func (*fakeOrbitHostStore) SetOrbitDeviceAuthToken(context.Context, string, string) error { return nil }

func (s *fakeOrbitHostStore) ValidateOrbitDeviceAuthToken(context.Context, string) (*hosts.Host, error) {
	return s.host, s.validateErr
}

type fakeOrbitSecretVerifier struct{ ok bool }

func (v fakeOrbitSecretVerifier) Verify(context.Context, agentauth.Agent, string) (bool, error) {
	return v.ok, nil
}

type fakePrimaryUserStore struct{}

func (fakePrimaryUserStore) Upsert(context.Context, int64, string, hosts.PrimaryUserSource) error {
	return nil
}

type heartbeatCall struct {
	hostID  int64
	source  heartbeats.Source
	contact heartbeats.Contact
}

type fakeHeartbeatRecorder struct {
	calls []heartbeatCall
	err   error
}

func (r *fakeHeartbeatRecorder) Record(_ context.Context, hostID int64, source heartbeats.Source, contact heartbeats.Contact) error {
	r.calls = append(r.calls, heartbeatCall{hostID: hostID, source: source, contact: contact})
	return r.err
}

func newTestEnrollmentService(
	recorder heartbeatRecorder,
	hostStore *fakeOrbitHostStore,
) *EnrollmentService {
	return NewEnrollmentService(
		hostStore,
		fakeOrbitSecretVerifier{ok: true},
		fakePrimaryUserStore{},
		recorder,
		fakeRemediationStore{},
	)
}

type fakeRemediationStore struct{}

func (fakeRemediationStore) PendingRemediationExecutionIDs(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (fakeRemediationStore) ClaimRemediation(
	context.Context,
	int64,
	string,
) (*policies.ClaimedRemediation, error) {
	return nil, fault.ErrNotFound
}

func (fakeRemediationStore) RecordRemediationResult(
	context.Context,
	int64,
	policies.RemediationResult,
) error {
	return nil
}
