package protocol

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
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
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestOsqueryEnrollValidation(t *testing.T) {
	database, ctx := dbtest.Open(t)
	secretStore := agentauth.NewStore(database)
	hostStore := hosts.NewStore(database)
	const enrollSecret = "focused-osquery-secret-0123456789abcdef"
	if _, err := secretStore.Create(ctx, agentauth.AgentSecretCreate{
		Agent: agentauth.AgentOrbit,
		Value: enrollSecret,
	}); err != nil {
		t.Fatalf("create Orbit secret: %v", err)
	}

	router := chi.NewRouter()
	logger := slog.New(slog.DiscardHandler)
	NewServer(osquery.NewAgentService(osquery.Dependencies{
		HostStore:   hostStore,
		SecretStore: secretStore,
		Logger:      logger,
	}), logger).RegisterRoutes(router)

	t.Run("invalid enrollment secret", func(t *testing.T) {
		recorder := postOsqueryEnroll(t, router, osquery.EnrollRequest{
			EnrollSecret:   "wrong-secret",
			HostIdentifier: "invalid-secret-host",
		})
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
		}
	})

	t.Run("missing hardware UUID and host identifier", func(t *testing.T) {
		recorder := postOsqueryEnroll(t, router, osquery.EnrollRequest{
			EnrollSecret: enrollSecret,
			HostDetails:  map[string]map[string]string{"system_info": {}},
		})
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})

	t.Run("host identifier is the hardware UUID fallback", func(t *testing.T) {
		const hostIdentifier = "focused-osquery-fallback-host"
		recorder := postOsqueryEnroll(t, router, osquery.EnrollRequest{
			EnrollSecret:   enrollSecret,
			HostIdentifier: hostIdentifier,
			HostDetails: map[string]map[string]string{
				"system_info": {"hostname": "fallback-host"},
			},
		})
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		var response osquery.EnrollResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode enroll response: %v", err)
		}
		if response.NodeKey == "" || response.NodeInvalid {
			t.Fatalf(
				"node key present/node_invalid = %t/%t, want true/false",
				response.NodeKey != "",
				response.NodeInvalid,
			)
		}
		host, err := hostStore.GetByOsqueryNodeKey(ctx, response.NodeKey)
		if err != nil {
			t.Fatalf("load enrolled host: %v", err)
		}
		if host.Hardware.UUID != hostIdentifier {
			t.Fatalf("hardware UUID = %q, want host identifier fallback %q", host.Hardware.UUID, hostIdentifier)
		}
	})
}

func postOsqueryEnroll(t *testing.T, router http.Handler, body osquery.EnrollRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode enroll request: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, osqueryPath+"/enroll", strings.NewReader(string(payload)))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func oversizedJSON(limit int64) string {
	return `{"padding":"` + strings.Repeat("x", int(limit)) + `"}`
}
