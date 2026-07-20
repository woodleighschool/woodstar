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

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
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
	enrollErr error
}

func (s *stubAgentService) Enroll(context.Context, osquery.EnrollRequest) (string, error) {
	return "node-key", s.enrollErr
}

func (*stubAgentService) Config(context.Context, string, string) (osquery.ConfigResponse, error) {
	return osquery.ConfigResponse{}, nil
}

func (*stubAgentService) DistributedRead(
	context.Context,
	string,
	string,
) (osquery.DistributedReadResponse, error) {
	return osquery.DistributedReadResponse{}, nil
}

func (*stubAgentService) DistributedWrite(
	context.Context,
	osquery.DistributedWriteRequest,
	string,
) (osquery.DistributedWriteResponse, error) {
	return osquery.DistributedWriteResponse{}, nil
}

func (*stubAgentService) Log(
	context.Context,
	string,
	string,
	osquery.LogRequest,
) (osquery.LogResponse, error) {
	return osquery.LogResponse{}, nil
}

func oversizedJSON(limit int64) string {
	return `{"padding":"` + strings.Repeat("x", int(limit)) + `"}`
}
