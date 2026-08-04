package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/osquery"
)

func TestOsqueryRoutesRejectMalformedAndOversizedJSON(t *testing.T) {
	router := chi.NewRouter()
	NewServer(nil, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "trailing JSON",
			path:       osqueryPath + "/config",
			body:       `{"node_key":"key"}{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "config over one MiB",
			path:       osqueryPath + "/config",
			body:       oversizedJSON(osqueryRequestMaxBytes),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "distributed write over five MiB",
			path:       osqueryPath + "/distributed/write",
			body:       oversizedJSON(osqueryDistributedWriteMaxBytes),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "log over ten MiB",
			path:       osqueryPath + "/log",
			body:       oversizedJSON(osqueryLogMaxBytes),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, tt.path, strings.NewReader(tt.body))
			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestOsqueryEnrollMapsServiceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "invalid enrollment secret", err: agentauth.ErrInvalidSecret, wantStatus: http.StatusUnauthorized},
		{name: "missing hardware UUID", err: enrollment.ErrMissingHardwareUUID, wantStatus: http.StatusBadRequest},
		{name: "service failure", err: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			router := chi.NewRouter()
			NewServer(&stubAgentService{enrollErr: tt.err}, slog.New(slog.DiscardHandler)).RegisterRoutes(router)
			recorder := postOsqueryEnroll(t, router, osquery.EnrollRequest{
				EnrollSecret:   "enroll-secret",
				HostIdentifier: "host-identifier",
			})
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestOsqueryConfigPassesRequestContact(t *testing.T) {
	t.Parallel()

	service := &stubAgentService{}
	router := chi.NewRouter()
	router.Use(chimiddleware.ClientIPFromRemoteAddr)
	NewServer(service, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	body, err := json.Marshal(osquery.ConfigRequest{NodeKey: "node-key"})
	if err != nil {
		t.Fatalf("marshal config request: %v", err)
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, osqueryPath+"/config", strings.NewReader(string(body)))
	req.RemoteAddr = "203.0.113.42:1234"
	req.Header.Set("User-Agent", "osquery/5.12.1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	want := heartbeats.Contact{RemoteIP: "203.0.113.42", UserAgent: "osquery/5.12.1"}
	if service.configContact != want {
		t.Fatalf("contact = %#v, want %#v", service.configContact, want)
	}
}

func postOsqueryEnroll(t *testing.T, router http.Handler, body osquery.EnrollRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode enroll request: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, osqueryPath+"/enroll", strings.NewReader(string(payload)))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

type stubAgentService struct {
	enrollErr     error
	configContact heartbeats.Contact
}

func (s *stubAgentService) Enroll(context.Context, osquery.EnrollRequest, heartbeats.Contact) (string, error) {
	return "node-key", s.enrollErr
}

func (s *stubAgentService) Config(_ context.Context, _ string, contact heartbeats.Contact) (osquery.ConfigResponse, error) {
	s.configContact = contact
	return osquery.ConfigResponse{}, nil
}

func (*stubAgentService) DistributedRead(
	context.Context,
	string,
	heartbeats.Contact,
) (osquery.DistributedReadResponse, error) {
	return osquery.DistributedReadResponse{}, nil
}

func (*stubAgentService) DistributedWrite(
	context.Context,
	osquery.DistributedWriteRequest,
	heartbeats.Contact,
) (osquery.DistributedWriteResponse, error) {
	return osquery.DistributedWriteResponse{}, nil
}

func (*stubAgentService) Log(
	context.Context,
	string,
	heartbeats.Contact,
	osquery.LogRequest,
) (osquery.LogResponse, error) {
	return osquery.LogResponse{}, nil
}

func oversizedJSON(limit int64) string {
	return `{"padding":"` + strings.Repeat("x", int(limit)) + `"}`
}
