package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
)

func TestAgentContract(t *testing.T) {
	databaseURL := integrationDatabaseURL(t)
	ctx := context.Background()

	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(db.Close)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	hardwareUUID := "woodstar-contract-" + suffix
	adminEmail := "contract-" + suffix + "@example.test"
	deviceEmail := "student-" + suffix + "@example.test"
	softwareName := "Example App " + suffix
	softwareBundleID := "com.example.contract." + suffix

	var secretValue string
	t.Cleanup(func() {
		cleanupContractRows(context.Background(), t, db, contractCleanup{
			HardwareUUID: hardwareUUID,
			AdminEmail:   adminEmail,
			SecretValue:  secretValue,
			BundleID:     softwareBundleID,
		})
	})

	deps, users := contractDependencies(t, db)
	router := NewServer(deps).routes()

	secret, err := deps.SecretStore.Create(ctx, models.SecretOrbit)
	if err != nil {
		t.Fatalf("create enroll secret: %v", err)
	}
	secretValue = secret.Value

	adminCookie := loginContractAdmin(t, users, deps.AuthService, deps.SessionManager, adminEmail)
	orbitNodeKey := orbitEnroll(t, router, secret.Value, hardwareUUID)
	assertOrbitPing(t, router)
	putDeviceMapping(t, router, orbitNodeKey, deviceEmail)

	osqueryNodeKey := osqueryEnroll(t, router, secret.Value, hardwareUUID)
	detailQueries := osqueryDistributedRead(t, router, osqueryNodeKey)
	assertDetailQueries(t, detailQueries)
	osqueryDistributedWrite(t, router, osqueryNodeKey, softwareName, softwareBundleID)

	hostID := assertAdminHost(t, router, adminCookie, hardwareUUID, deviceEmail)
	assertAdminHostSoftware(t, router, adminCookie, hostID, softwareName, softwareBundleID)
	assertAdminSoftware(t, router, adminCookie, softwareName, softwareBundleID)
}

func integrationDatabaseURL(t *testing.T) string {
	t.Helper()
	if os.Getenv("WOODSTAR_INTEGRATION") == "" {
		t.Skip("WOODSTAR_INTEGRATION is not set")
	}
	databaseURL := os.Getenv("WOODSTAR_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("WOODSTAR_TEST_DATABASE_URL is not set")
	}
	return databaseURL
}

func contractDependencies(t *testing.T, db *database.DB) (Dependencies, *models.UserStore) {
	t.Helper()

	users := models.NewUserStore(db)
	hosts := models.NewHostStore(db)
	deviceMappings := models.NewDeviceMappingStore(db)
	secrets := models.NewSecretStore(db)
	software := models.NewSoftwareStore(db)

	sessionManager := scs.New()
	sessionManager.Store = memstore.New()

	authService := auth.NewService(users, sessionManager)

	return Dependencies{
		Config: config.Config{
			PublicURL:     "http://localhost:8080",
			SessionSecret: strings.Repeat("s", 32),
		},
		DB:             db,
		Version:        "test",
		AuthService:    authService,
		SessionManager: sessionManager,
		HostStore:      hosts,
		DeviceMappings: deviceMappings,
		SecretStore:    secrets,
		SoftwareStore:  software,
		OrbitService:   orbit.NewService(hosts, secrets, deviceMappings),
		OsqueryService: osquery.NewService(hosts, software, secrets),
	}, users
}

type contractCleanup struct {
	HardwareUUID string
	AdminEmail   string
	SecretValue  string
	BundleID     string
}

func cleanupContractRows(ctx context.Context, t *testing.T, db *database.DB, cleanup contractCleanup) {
	t.Helper()
	statements := []struct {
		sql  string
		args []any
	}{
		{sql: `DELETE FROM hosts WHERE hardware_uuid = $1`, args: []any{cleanup.HardwareUUID}},
		{sql: `DELETE FROM users WHERE email = $1`, args: []any{cleanup.AdminEmail}},
		{sql: `DELETE FROM secrets WHERE value = $1`, args: []any{cleanup.SecretValue}},
		{sql: `DELETE FROM software WHERE bundle_identifier = $1`, args: []any{cleanup.BundleID}},
	}
	for _, stmt := range statements {
		if _, err := db.Pool().Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatalf("cleanup contract rows: %v", err)
		}
	}
}

func loginContractAdmin(
	t *testing.T,
	users *models.UserStore,
	authService *auth.Service,
	sessionManager *scs.SessionManager,
	email string,
) *http.Cookie {
	t.Helper()

	const password = "contract-password"
	ctx, err := sessionManager.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash contract password: %v", err)
	}
	user, err := users.Create(ctx, models.CreateUserParams{
		Email:        email,
		Name:         "Contract Admin",
		PasswordHash: hash,
		Role:         models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create contract admin: %v", err)
	}
	if user == nil {
		t.Fatal("create contract admin returned nil user")
	}
	if _, err := authService.Login(ctx, email, password); err != nil {
		t.Fatalf("login contract admin: %v", err)
	}
	token, _, err := sessionManager.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return &http.Cookie{Name: sessionManager.Cookie.Name, Value: token}
}

func orbitEnroll(t *testing.T, router http.Handler, secret string, hardwareUUID string) string {
	t.Helper()
	var body struct {
		OrbitNodeKey string `json:"orbit_node_key"`
	}
	doJSON(t, router, http.MethodPost, "/api/fleet/orbit/enroll", map[string]any{
		"enroll_secret":   secret,
		"hardware_uuid":   hardwareUUID,
		"hardware_serial": "C02CONTRACT",
		"hostname":        "contract-mac",
		"computer_name":   "Contract Mac",
		"hardware_model":  "Mac15,8",
		"platform":        "darwin",
		"platform_like":   "darwin",
	}, nil, &body)
	if body.OrbitNodeKey == "" {
		t.Fatal("orbit enroll returned empty node key")
	}
	return body.OrbitNodeKey
}

func assertOrbitPing(t *testing.T, router http.Handler) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/api/fleet/orbit/ping", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("orbit ping status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Fleet-Capabilities"); !strings.Contains(got, "end_user_email") {
		t.Fatalf("orbit capabilities = %q, want end_user_email", got)
	}
}

func putDeviceMapping(t *testing.T, router http.Handler, orbitNodeKey string, email string) {
	t.Helper()
	doJSON(t, router, http.MethodPut, "/api/fleet/orbit/device_mapping", map[string]any{
		"orbit_node_key": orbitNodeKey,
		"email":          email,
	}, nil, nil)
}

func osqueryEnroll(t *testing.T, router http.Handler, secret string, hardwareUUID string) string {
	t.Helper()
	var body struct {
		NodeKey string `json:"node_key"`
	}
	doJSON(t, router, http.MethodPost, "/api/v1/osquery/enroll", map[string]any{
		"enroll_secret":   secret,
		"host_identifier": hardwareUUID,
		"host_details": map[string]map[string]string{
			"system_info": {
				"uuid":               hardwareUUID,
				"hostname":           "contract-mac",
				"computer_name":      "Contract Mac",
				"hardware_serial":    "C02CONTRACT",
				"hardware_model":     "Mac15,8",
				"hardware_vendor":    "Apple Inc.",
				"cpu_brand":          "Apple M4",
				"cpu_logical_cores":  "10",
				"cpu_physical_cores": "10",
				"physical_memory":    "68719476736",
			},
			"osquery_info": {"version": "5.22.1"},
			"os_version": {
				"name":          "macOS",
				"version":       "26.5",
				"build":         "25F5068a",
				"platform":      "darwin",
				"platform_like": "darwin",
			},
			"platform_info": {"extra": "Darwin Kernel Version 25.5.0"},
		},
	}, nil, &body)
	if body.NodeKey == "" {
		t.Fatal("osquery enroll returned empty node key")
	}
	return body.NodeKey
}

func osqueryDistributedRead(t *testing.T, router http.Handler, nodeKey string) map[string]string {
	t.Helper()
	var body struct {
		NodeInvalid bool              `json:"node_invalid"`
		Queries     map[string]string `json:"queries"`
	}
	doJSON(t, router, http.MethodPost, "/api/v1/osquery/distributed/read", map[string]any{
		"node_key": nodeKey,
	}, nil, &body)
	if body.NodeInvalid {
		t.Fatal("distributed read returned node_invalid")
	}
	return body.Queries
}

func assertDetailQueries(t *testing.T, queries map[string]string) {
	t.Helper()
	for _, name := range []string{"os_version", "system_info", "osquery_info", "software_macos"} {
		if queries[name] == "" {
			t.Fatalf("missing detail query %q", name)
		}
	}
}

func osqueryDistributedWrite(t *testing.T, router http.Handler, nodeKey string, softwareName string, bundleID string) {
	t.Helper()
	doJSON(t, router, http.MethodPost, "/api/v1/osquery/distributed/write", map[string]any{
		"node_key": nodeKey,
		"queries": map[string][]map[string]string{
			"os_version": {{
				"name":          "macOS",
				"version":       "26.5",
				"build":         "25F5068a",
				"platform":      "darwin",
				"platform_like": "darwin",
			}},
			"system_info": {{
				"hostname":           "contract-mac",
				"computer_name":      "Contract Mac",
				"hardware_serial":    "C02CONTRACT",
				"hardware_model":     "Mac15,8",
				"hardware_vendor":    "Apple Inc.",
				"cpu_brand":          "Apple M4",
				"cpu_logical_cores":  "10",
				"cpu_physical_cores": "10",
				"physical_memory":    "68719476736",
			}},
			"osquery_info": {{"version": "5.22.1"}},
			"software_macos": {{
				"name":              softwareName,
				"version":           "1.2.3",
				"source":            "apps",
				"bundle_identifier": bundleID,
				"path":              "/Applications/Example App.app",
				"last_opened_time":  "1777435200",
			}},
		},
		"statuses": map[string]int{
			"os_version":     0,
			"system_info":    0,
			"osquery_info":   0,
			"software_macos": 0,
		},
	}, nil, nil)
}

func assertAdminHost(
	t *testing.T,
	router http.Handler,
	adminCookie *http.Cookie,
	hardwareUUID string,
	deviceEmail string,
) string {
	t.Helper()
	var hosts []struct {
		ID             string                  `json:"id"`
		HardwareUUID   string                  `json:"hardware_uuid"`
		DisplayName    string                  `json:"display_name"`
		PhysicalMemory int64                   `json:"physical_memory"`
		DeviceMappings []contractDeviceMapping `json:"device_mappings"`
	}
	doJSON(t, router, http.MethodGet, "/api/hosts", nil, adminCookie, &hosts)
	for _, host := range hosts {
		if host.HardwareUUID != hardwareUUID {
			continue
		}
		if host.DisplayName != "Contract Mac" {
			t.Fatalf("host display_name = %q, want Contract Mac", host.DisplayName)
		}
		if host.PhysicalMemory != 68719476736 {
			t.Fatalf("host physical_memory = %d, want 68719476736", host.PhysicalMemory)
		}
		assertDeviceMapping(t, host.DeviceMappings, deviceEmail)
		return host.ID
	}
	t.Fatalf("host %q not found in admin list", hardwareUUID)
	return ""
}

type contractDeviceMapping struct {
	Email  string `json:"email"`
	Source string `json:"source"`
}

type contractSoftwareTitle struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Source           string `json:"source"`
	BundleIdentifier string `json:"bundle_identifier"`
}

type contractSoftwareTitleWithCount struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Source           string `json:"source"`
	BundleIdentifier string `json:"bundle_identifier"`
	HostCount        int    `json:"host_count"`
}

func assertDeviceMapping(t *testing.T, mappings []contractDeviceMapping, email string) {
	t.Helper()
	for _, mapping := range mappings {
		if mapping.Email == email && mapping.Source == models.DeviceMappingSourceOrbitProfile {
			return
		}
	}
	t.Fatalf("device mapping %q with source %q not found", email, models.DeviceMappingSourceOrbitProfile)
}

func assertAdminHostSoftware(
	t *testing.T,
	router http.Handler,
	adminCookie *http.Cookie,
	hostID string,
	softwareName string,
	bundleID string,
) {
	t.Helper()
	var software []contractSoftwareTitle
	doJSON(t, router, http.MethodGet, "/api/hosts/"+hostID+"/software", nil, adminCookie, &software)
	assertExampleSoftware(t, software, softwareName, bundleID)
}

func assertAdminSoftware(
	t *testing.T,
	router http.Handler,
	adminCookie *http.Cookie,
	softwareName string,
	bundleID string,
) {
	t.Helper()
	var software []contractSoftwareTitleWithCount
	doJSON(t, router, http.MethodGet, "/api/software", nil, adminCookie, &software)
	for _, title := range software {
		if title.Name == softwareName && title.Version == "1.2.3" && title.Source == "apps" &&
			title.BundleIdentifier == bundleID && title.HostCount >= 1 {
			return
		}
	}
	t.Fatalf("%s title not found in admin software list", softwareName)
}

func assertExampleSoftware(t *testing.T, software []contractSoftwareTitle, softwareName string, bundleID string) {
	t.Helper()
	for _, title := range software {
		if title.Name == softwareName && title.Version == "1.2.3" && title.Source == "apps" &&
			title.BundleIdentifier == bundleID {
			return
		}
	}
	t.Fatalf("%s not found in host software list", softwareName)
}

func doJSON(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	payload any,
	cookie *http.Cookie,
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
	if cookie != nil {
		req.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s %s status = %d, want %d; body: %s", method, path, rec.Code, http.StatusOK, rec.Body.String())
	}
	if out != nil {
		if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
			t.Fatalf("decode %s %s response: %v; body: %s", method, path, err, rec.Body.String())
		}
	}
}
