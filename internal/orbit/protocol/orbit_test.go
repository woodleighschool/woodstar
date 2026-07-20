package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/orbit"
)

func TestPingReportsSupportedCapabilities(t *testing.T) {
	router := chi.NewRouter()
	NewServer(nil, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodHead, "/api/fleet/orbit/ping", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	got := rec.Header().Get(capabilitiesHeader)
	if got != orbitCapabilitiesValue {
		t.Fatalf("capabilities = %q, want %q", got, orbitCapabilitiesValue)
	}
}

func TestOrbitDeviceMappingMapsServiceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "malformed email", err: dbutil.ErrInvalidInput, wantStatus: http.StatusBadRequest},
		{name: "unknown node key", err: dbutil.ErrNotFound, wantStatus: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := &stubEnrollmentService{setPrimaryUserErr: tt.err}
			router := newOrbitRouter(service)
			doOrbitJSON(t, router, http.MethodPut, "/api/fleet/orbit/device_mapping", orbit.DeviceMappingRequest{
				OrbitNodeKey: "node-key",
				Email:        "student@example.test",
			}, tt.wantStatus)
		})
	}
}

func TestOrbitDeviceTokenMapsServiceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "invalid token", err: orbit.ErrInvalidDeviceAuthToken, wantStatus: http.StatusBadRequest},
		{name: "unknown node key", err: dbutil.ErrNotFound, wantStatus: http.StatusUnauthorized},
		{name: "duplicate token", err: dbutil.ErrAlreadyExists, wantStatus: http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := &stubEnrollmentService{setDeviceAuthTokenErr: tt.err}
			doOrbitJSON(t, newOrbitRouter(service), http.MethodPost, "/api/fleet/orbit/device_token", orbit.DeviceTokenRequest{
				OrbitNodeKey:    "node-key",
				DeviceAuthToken: "device-token",
			}, tt.wantStatus)
		})
	}
}

func TestOrbitDevicePingRejectsUnknownToken(t *testing.T) {
	t.Parallel()

	service := &stubEnrollmentService{validateDeviceAuthTokenErr: dbutil.ErrNotFound}
	doOrbitJSON(
		t,
		newOrbitRouter(service),
		http.MethodHead,
		"/api/latest/fleet/device/471f74c8-4192-444b-8c77-da229df57f29/ping",
		nil,
		http.StatusUnauthorized,
	)
}

func TestOrbitEnrollRejectsInvalidSecret(t *testing.T) {
	t.Parallel()

	service := &stubEnrollmentService{enrollErr: agentauth.ErrInvalidSecret}
	doOrbitJSON(
		t, newOrbitRouter(service), http.MethodPost, "/api/fleet/orbit/enroll", orbit.EnrollRequest{ //nolint:gosec // Intentionally invalid enrollment-secret fixture.
			EnrollSecret: "not-a-real-secret",
			HardwareUUID: "orbit-invalid-secret",
		}, http.StatusUnauthorized)
}

func TestOrbitRoutesRejectMalformedAndOversizedJSON(t *testing.T) {
	router := chi.NewRouter()
	NewServer(nil, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "trailing JSON",
			body:       `{"orbit_node_key":"key"}{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "oversized body",
			body:       `{"padding":"` + strings.Repeat("x", orbitRequestMaxBytes) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/fleet/orbit/config", strings.NewReader(tt.body))
			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

type stubEnrollmentService struct {
	enrollErr                  error
	configErr                  error
	setPrimaryUserErr          error
	setDeviceAuthTokenErr      error
	validateDeviceAuthTokenErr error
}

func (s *stubEnrollmentService) Enroll(
	context.Context,
	orbit.EnrollRequest,
) (*hosts.Host, string, error) {
	return &hosts.Host{}, "node-key", s.enrollErr
}

func (s *stubEnrollmentService) Config(context.Context, string) (orbit.ConfigResponse, error) {
	return orbit.ConfigResponse{}, s.configErr
}

func (s *stubEnrollmentService) SetPrimaryUser(context.Context, string, string) error {
	return s.setPrimaryUserErr
}

func (s *stubEnrollmentService) SetDeviceAuthToken(context.Context, string, string) error {
	return s.setDeviceAuthTokenErr
}

func (s *stubEnrollmentService) ValidateDeviceAuthToken(context.Context, string) error {
	return s.validateDeviceAuthTokenErr
}

func newOrbitRouter(service enrollmentService) http.Handler {
	router := chi.NewRouter()
	NewServer(service, slog.New(slog.DiscardHandler)).RegisterRoutes(router)
	return router
}

func doOrbitJSON(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	payload any,
	wantStatus int,
) {
	t.Helper()

	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequestWithContext(t.Context(), method, path, &body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d; body: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	if got := rec.Header().Get(capabilitiesHeader); !strings.Contains(got, "end_user_email") {
		t.Fatalf("capabilities header = %q, want end_user_email", got)
	}
	if got := rec.Header().Get(capabilitiesHeader); !strings.Contains(got, "token_rotation") {
		t.Fatalf("capabilities header = %q, want token_rotation", got)
	}
}
