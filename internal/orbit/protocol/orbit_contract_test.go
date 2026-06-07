package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/orbit"
)

func TestOrbitHTTPEnrollConfigAndUserAffinity(t *testing.T) {
	database, ctx := dbtest.Open(t)
	stores := newOrbitContractStores(database)
	router := newOrbitContractRouter(stores)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	hardwareUUID := "orbit-contract-" + suffix
	userEmail := "student-" + suffix + "@example.test"

	secret, err := stores.agentSecrets.Create(
		ctx,
		agentauth.AgentSecretCreate{Agent: agentauth.AgentOrbit, Value: "orbit-contract-secret-value-" + suffix},
	)
	if err != nil {
		t.Fatalf("create orbit agent secret: %v", err)
	}
	t.Cleanup(func() {
		cleanupOrbitContractRows(context.Background(), t, database, hardwareUUID, secret.Value)
	})

	var enrollBody orbit.EnrollResponse
	doOrbitJSON(t, router, http.MethodPost, "/api/fleet/orbit/enroll", orbit.EnrollRequest{
		EnrollSecret:   secret.Value,
		HardwareUUID:   hardwareUUID,
		HardwareSerial: "C02ORBIT",
		Hostname:       "orbit-mac",
		ComputerName:   "Orbit Mac",
		HardwareModel:  "Mac15,8",
	}, http.StatusOK, &enrollBody)
	if enrollBody.OrbitNodeKey == "" {
		t.Fatal("enroll returned empty orbit node key")
	}

	var configBody map[string]any
	doOrbitJSON(
		t,
		router,
		http.MethodPost,
		"/api/fleet/orbit/config",
		orbit.ConfigRequest(enrollBody),
		http.StatusOK,
		&configBody,
	)
	if len(configBody) != 0 {
		t.Fatalf("config body = %#v, want empty object", configBody)
	}

	doOrbitJSON(t, router, http.MethodPut, "/api/fleet/orbit/device_mapping", orbit.DeviceMappingRequest{
		OrbitNodeKey: enrollBody.OrbitNodeKey,
		Email:        userEmail,
	}, http.StatusOK, nil)

	host, err := stores.hosts.GetByOrbitNodeKey(ctx, enrollBody.OrbitNodeKey)
	if err != nil {
		t.Fatalf("get host by orbit node key: %v", err)
	}
	detail, err := stores.hosts.LoadDetail(ctx, host)
	if err != nil {
		t.Fatalf("load host detail: %v", err)
	}
	if !hasUserAffinityMapping(detail.UserAffinity.Mappings, userEmail) {
		t.Fatalf("user affinity mapping %q not found: %#v", userEmail, detail.UserAffinity.Mappings)
	}
}

func TestOrbitHTTPRejectsInvalidEnrollSecret(t *testing.T) {
	database, _ := dbtest.Open(t)
	stores := newOrbitContractStores(database)
	router := newOrbitContractRouter(stores)

	doOrbitJSON(t, router, http.MethodPost, "/api/fleet/orbit/enroll", orbit.EnrollRequest{
		EnrollSecret: "not-a-real-secret",
		HardwareUUID: "orbit-invalid-secret",
	}, http.StatusUnauthorized, nil)
}

type orbitContractStores struct {
	hosts          *hosts.Store
	userAffinities *hosts.UserAffinityStore
	agentSecrets   *agentauth.Store
}

func newOrbitContractStores(database *database.DB) orbitContractStores {
	return orbitContractStores{
		hosts:          hosts.NewStore(database),
		userAffinities: hosts.NewUserAffinityStore(database),
		agentSecrets:   agentauth.NewStore(database),
	}
}

func newOrbitContractRouter(stores orbitContractStores) http.Handler {
	router := chi.NewRouter()
	RegisterOrbitRoutes(
		router,
		orbit.NewEnrollmentService(stores.hosts, stores.agentSecrets, stores.userAffinities),
		slog.New(slog.DiscardHandler),
	)
	return router
}

func hasUserAffinityMapping(mappings []hosts.HostUserAffinityMapping, email string) bool {
	for _, mapping := range mappings {
		if mapping.Email == email && mapping.Source == hosts.UserAffinitySourceOrbitProfile {
			return true
		}
	}
	return false
}

func cleanupOrbitContractRows(
	ctx context.Context,
	t *testing.T,
	database *database.DB,
	hardwareUUID string,
	secretValue string,
) {
	t.Helper()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{sql: `DELETE FROM hosts WHERE hardware_uuid = $1`, args: []any{hardwareUUID}},
		{sql: `DELETE FROM agent_secrets WHERE value = $1`, args: []any{secretValue}},
	} {
		if _, err := database.Pool().Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatalf("cleanup orbit contract rows: %v", err)
		}
	}
}

func doOrbitJSON(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	payload any,
	wantStatus int,
	out any,
) {
	t.Helper()

	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &body)
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
	if out != nil {
		if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
			t.Fatalf("decode %s %s response: %v; body: %s", method, path, err, rec.Body.String())
		}
	}
}
