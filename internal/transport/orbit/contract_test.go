package orbit

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

	"github.com/woodleighschool/woodstar/internal/db"
	"github.com/woodleighschool/woodstar/internal/db/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/models"
	coreorbit "github.com/woodleighschool/woodstar/internal/orbit"
)

func TestOrbitHTTPEnrollConfigAndDeviceMapping(t *testing.T) {
	database, ctx := dbtest.Open(t)
	stores := newOrbitContractStores(database)
	router := newOrbitContractRouter(stores)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	hardwareUUID := "orbit-contract-" + suffix
	deviceEmail := "student-" + suffix + "@example.test"

	secret, err := stores.secrets.Create(ctx, models.SecretOrbit)
	if err != nil {
		t.Fatalf("create enroll secret: %v", err)
	}
	t.Cleanup(func() {
		cleanupOrbitContractRows(context.Background(), t, database, hardwareUUID, secret.Value)
	})

	var enrollBody coreorbit.EnrollResponse
	doOrbitJSON(t, router, http.MethodPost, "/api/fleet/orbit/enroll", coreorbit.EnrollRequest{
		EnrollSecret:   secret.Value,
		HardwareUUID:   hardwareUUID,
		HardwareSerial: "C02ORBIT",
		Hostname:       "orbit-mac",
		ComputerName:   "Orbit Mac",
		HardwareModel:  "Mac15,8",
		Platform:       "darwin",
		PlatformLike:   "darwin",
	}, http.StatusOK, &enrollBody)
	if enrollBody.OrbitNodeKey == "" {
		t.Fatal("enroll returned empty orbit node key")
	}

	var configBody coreorbit.ConfigResponse
	doOrbitJSON(t, router, http.MethodPost, "/api/fleet/orbit/config", coreorbit.ConfigRequest{
		OrbitNodeKey: enrollBody.OrbitNodeKey,
	}, http.StatusOK, &configBody)
	if string(configBody.Flags) != "{}" {
		t.Fatalf("config flags = %s, want {}", configBody.Flags)
	}

	doOrbitJSON(t, router, http.MethodPut, "/api/fleet/orbit/device_mapping", coreorbit.DeviceMappingRequest{
		OrbitNodeKey: enrollBody.OrbitNodeKey,
		Email:        deviceEmail,
	}, http.StatusOK, nil)

	host, err := stores.hosts.GetByOrbitNodeKey(ctx, enrollBody.OrbitNodeKey)
	if err != nil {
		t.Fatalf("get host by orbit node key: %v", err)
	}
	mappings, err := stores.deviceMappings.ListForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("list device mappings: %v", err)
	}
	if !hasDeviceMapping(mappings, deviceEmail) {
		t.Fatalf("device mapping %q not found: %#v", deviceEmail, mappings)
	}
}

func TestOrbitHTTPRejectsInvalidEnrollSecret(t *testing.T) {
	database, _ := dbtest.Open(t)
	stores := newOrbitContractStores(database)
	router := newOrbitContractRouter(stores)

	doOrbitJSON(t, router, http.MethodPost, "/api/fleet/orbit/enroll", coreorbit.EnrollRequest{
		EnrollSecret: "not-a-real-secret",
		HardwareUUID: "orbit-invalid-secret",
	}, http.StatusUnauthorized, nil)
}

type orbitContractStores struct {
	hosts          *hosts.HostStore
	deviceMappings *hosts.DeviceMappingStore
	secrets        *models.SecretStore
}

func newOrbitContractStores(database *db.DB) orbitContractStores {
	return orbitContractStores{
		hosts:          hosts.NewHostStore(database),
		deviceMappings: hosts.NewDeviceMappingStore(database),
		secrets:        models.NewSecretStore(database),
	}
}

func newOrbitContractRouter(stores orbitContractStores) http.Handler {
	router := chi.NewRouter()
	RegisterRoutes(
		router,
		coreorbit.NewService(stores.hosts, stores.secrets, stores.deviceMappings),
		slog.New(slog.DiscardHandler),
	)
	return router
}

func hasDeviceMapping(mappings []hosts.HostDeviceMapping, email string) bool {
	for _, mapping := range mappings {
		if mapping.Email == email && mapping.Source == hosts.DeviceMappingSourceOrbitProfile {
			return true
		}
	}
	return false
}

func cleanupOrbitContractRows(
	ctx context.Context,
	t *testing.T,
	database *db.DB,
	hardwareUUID string,
	secretValue string,
) {
	t.Helper()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{sql: `DELETE FROM hosts WHERE hardware_uuid = $1`, args: []any{hardwareUUID}},
		{sql: `DELETE FROM secrets WHERE value = $1`, args: []any{secretValue}},
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
