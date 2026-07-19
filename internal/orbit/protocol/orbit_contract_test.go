package protocol

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/orbit"
)

func TestOrbitDeviceMappingPersistsProfilePrimaryUser(t *testing.T) {
	database, ctx := dbtest.Open(t)
	stores := newOrbitContractStores(database)
	router := newOrbitContractRouter(stores)

	const (
		nodeKey   = "orbit-contract-node-key"
		userEmail = "student@example.test"
	)
	_, err := stores.hosts.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "orbit-contract-mapping"},
		OrbitNodeKey: nodeKey,
	})
	if err != nil {
		t.Fatalf("seed Orbit host: %v", err)
	}

	doOrbitJSON(t, router, http.MethodPut, "/api/fleet/orbit/device_mapping", orbit.DeviceMappingRequest{
		OrbitNodeKey: nodeKey,
		Email:        userEmail,
	}, http.StatusOK)

	host, err := stores.hosts.GetByOrbitNodeKey(ctx, nodeKey)
	if err != nil {
		t.Fatalf("get host by orbit node key: %v", err)
	}
	detail, err := stores.hosts.LoadDetail(ctx, host)
	if err != nil {
		t.Fatalf("load host detail: %v", err)
	}
	if !hasPrimaryUserSource(detail.PrimaryUserSources, userEmail) {
		t.Fatalf("primary user source %q not found: %#v", userEmail, detail.PrimaryUserSources)
	}
}

func TestOrbitDeviceMappingRejectsMalformedEmail(t *testing.T) {
	database, ctx := dbtest.Open(t)
	stores := newOrbitContractStores(database)
	router := newOrbitContractRouter(stores)

	const nodeKey = "orbit-contract-node-key"
	_, err := stores.hosts.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "orbit-contract-malformed-email"},
		OrbitNodeKey: nodeKey,
	})
	if err != nil {
		t.Fatalf("seed Orbit host: %v", err)
	}

	doOrbitJSON(t, router, http.MethodPut, "/api/fleet/orbit/device_mapping", orbit.DeviceMappingRequest{
		OrbitNodeKey: nodeKey,
		Email:        "not-an-email",
	}, http.StatusBadRequest)
}

func TestOrbitDeviceMappingRejectsUnknownNodeKey(t *testing.T) {
	database, _ := dbtest.Open(t)
	stores := newOrbitContractStores(database)
	router := newOrbitContractRouter(stores)

	doOrbitJSON(t, router, http.MethodPut, "/api/fleet/orbit/device_mapping", orbit.DeviceMappingRequest{
		OrbitNodeKey: "not-a-node-key",
		Email:        "valid@example.test",
	}, http.StatusUnauthorized)
}

func TestOrbitDevicePingRejectsUnknownToken(t *testing.T) {
	database, _ := dbtest.Open(t)
	stores := newOrbitContractStores(database)
	router := newOrbitContractRouter(stores)

	doOrbitJSON(
		t,
		router,
		http.MethodHead,
		"/api/latest/fleet/device/471f74c8-4192-444b-8c77-da229df57f29/ping",
		nil,
		http.StatusUnauthorized,
	)
}

func TestOrbitHTTPRejectsInvalidEnrollSecret(t *testing.T) {
	database, _ := dbtest.Open(t)
	stores := newOrbitContractStores(database)
	router := newOrbitContractRouter(stores)

	doOrbitJSON(t, router, http.MethodPost, "/api/fleet/orbit/enroll", orbit.EnrollRequest{
		EnrollSecret: "not-a-real-secret",
		HardwareUUID: "orbit-invalid-secret",
	}, http.StatusUnauthorized)
}

type orbitContractStores struct {
	hosts        *hosts.Store
	primaryUsers *hosts.PrimaryUserStore
	agentSecrets *agentauth.Store
}

func newOrbitContractStores(database *database.DB) orbitContractStores {
	return orbitContractStores{
		hosts:        hosts.NewStore(database),
		primaryUsers: hosts.NewPrimaryUserStore(database),
		agentSecrets: agentauth.NewStore(database),
	}
}

func newOrbitContractRouter(stores orbitContractStores) http.Handler {
	router := chi.NewRouter()
	NewServer(
		orbit.NewEnrollmentService(stores.hosts, stores.agentSecrets, stores.primaryUsers),
		slog.New(slog.DiscardHandler),
	).RegisterRoutes(router)
	return router
}

func hasPrimaryUserSource(sources []hosts.HostPrimaryUserSource, email string) bool {
	for _, source := range sources {
		if source.Email == email && source.Source == hosts.PrimaryUserSourceOrbitProfile {
			return true
		}
	}
	return false
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
	if got := rec.Header().Get(capabilitiesHeader); !strings.Contains(got, "token_rotation") {
		t.Fatalf("capabilities header = %q, want token_rotation", got)
	}
}
