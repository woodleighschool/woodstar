package transport

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

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/db"
	"github.com/woodleighschool/woodstar/internal/db/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
	queryinfra "github.com/woodleighschool/woodstar/internal/queries"
)

func TestAgentContract(t *testing.T) {
	database, ctx := dbtest.Open(t)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	hardwareUUID := "woodstar-contract-" + suffix
	adminEmail := "contract-" + suffix + "@example.test"
	deviceEmail := "student-" + suffix + "@example.test"
	softwareName := "Example App " + suffix
	softwareBundleID := "com.example.contract." + suffix

	var secretValue string
	t.Cleanup(func() {
		cleanupContractRows(context.Background(), t, database, contractCleanup{
			HardwareUUID: hardwareUUID,
			AdminEmail:   adminEmail,
			SecretValue:  secretValue,
			BundleID:     softwareBundleID,
		})
	})

	deps, users := contractDependencies(t, database)
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

func contractDependencies(t *testing.T, db *db.DB) (Dependencies, *models.UserStore) {
	t.Helper()

	users := models.NewUserStore(db)
	hostStore := hosts.NewHostStore(db)
	deviceMappings := hosts.NewDeviceMappingStore(db)
	secrets := models.NewSecretStore(db)
	software := models.NewSoftwareStore(db)
	labelStore := labels.NewLabelStore(db)
	queryStore := queryinfra.NewQueryStore(db)
	checkStore := queryinfra.NewCheckStore(db)
	hub := queryinfra.NewHub()
	liveQueries := queryinfra.NewLiveQueryManager(hub, time.Minute)

	sessionManager := scs.New()
	sessionManager.Store = memstore.New()

	authService := auth.NewService(users, sessionManager)
	logger := slog.New(slog.DiscardHandler)

	return Dependencies{
		Config: config.Config{
			PublicURL:     "http://localhost:8080",
			SessionSecret: strings.Repeat("s", 32),
		},
		DB:               db,
		Version:          "test",
		Logger:           logger,
		AuthService:      authService,
		SessionManager:   sessionManager,
		HostStore:        hostStore,
		DeviceMappings:   deviceMappings,
		SecretStore:      secrets,
		SoftwareStore:    software,
		LabelStore:       labelStore,
		QueryStore:       queryStore,
		CheckStore:       checkStore,
		LiveQueryManager: liveQueries,
		OrbitService:     orbit.NewService(hostStore, secrets, deviceMappings),
		OsqueryService: osquery.NewService(
			hostStore,
			software,
			labelStore,
			queryStore,
			checkStore,
			liveQueries,
			secrets,
			logger.With("component", "osquery"),
		),
	}, users
}

type contractCleanup struct {
	HardwareUUID string
	AdminEmail   string
	SecretValue  string
	BundleID     string
}

func cleanupContractRows(ctx context.Context, t *testing.T, db *db.DB, cleanup contractCleanup) {
	t.Helper()
	statements := []struct {
		sql  string
		args []any
	}{
		{sql: `DELETE FROM hosts WHERE hardware_uuid = $1`, args: []any{cleanup.HardwareUUID}},
		{sql: `DELETE FROM users WHERE email = $1`, args: []any{cleanup.AdminEmail}},
		{sql: `DELETE FROM secrets WHERE value = $1`, args: []any{cleanup.SecretValue}},
		{sql: `DELETE FROM software_titles WHERE bundle_identifier = $1`, args: []any{cleanup.BundleID}},
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
	for _, suffix := range []string{
		"os_version",
		"system_info",
		"osquery_info",
		"osquery_flags",
		"orbit_info",
		"uptime",
		"root_disk",
		"primary_interface",
		"users",
		"software_macos",
		"software_macos_codesign",
		"software_macos_executable_sha256",
	} {
		name := "woodstar_detail_query_" + suffix
		if queries[name] == "" {
			t.Fatalf("missing detail query %q", name)
		}
	}
}

func osqueryDistributedWrite(t *testing.T, router http.Handler, nodeKey string, softwareName string, bundleID string) {
	t.Helper()
	const prefix = "woodstar_detail_query_"
	queries := map[string][]map[string]string{
		prefix + "os_version": {{
			"name":          "macOS",
			"version":       "26.5",
			"build":         "25F5068a",
			"platform":      "darwin",
			"platform_like": "darwin",
		}},
		prefix + "system_info": {{
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
		prefix + "osquery_info": {{"version": "5.22.1"}},
		prefix + "osquery_flags": {
			{"name": "distributed_interval", "value": "15"},
			{"name": "config_refresh", "value": "60"},
		},
		prefix + "orbit_info": {{"version": "1.47.0"}},
		prefix + "uptime":     {{"total_seconds": "3600"}},
		prefix + "root_disk":  {{"bytes_available": "1073741824", "bytes_total": "4294967296"}},
		prefix + "primary_interface": {{
			"primary_ip":  "192.168.1.10",
			"primary_mac": "aa:bb:cc:dd:ee:ff",
		}},
		prefix + "users": {{
			"uid":         "501",
			"username":    "contract",
			"type":        "local",
			"description": "Contract User",
			"directory":   "/Users/contract",
			"shell":       "/bin/zsh",
		}},
		prefix + "software_macos": {{
			"name":              softwareName,
			"version":           "1.2.3",
			"source":            "apps",
			"bundle_identifier": bundleID,
			"installed_path":    "/Applications/Example App.app",
			"last_opened_at":    "1777435200.5",
		}},
		prefix + "software_macos_codesign": {{
			"path":            "/Applications/Example App.app",
			"team_identifier": "ABCD123456",
			"cdhash_sha256":   "cdhash",
		}},
		prefix + "software_macos_executable_sha256": {{
			"path":              "/Applications/Example App.app",
			"executable_sha256": "executable-hash",
			"executable_path":   "/Applications/Example App.app/Contents/MacOS/Example",
		}},
	}
	statuses := make(map[string]int, len(queries))
	for name := range queries {
		statuses[name] = 0
	}

	doJSON(t, router, http.MethodPost, "/api/v1/osquery/distributed/write", map[string]any{
		"node_key": nodeKey,
		"queries":  queries,
		"statuses": statuses,
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
	var body struct {
		Items []struct {
			ID             int64                   `json:"id"`
			HardwareUUID   string                  `json:"hardware_uuid"`
			DisplayName    string                  `json:"display_name"`
			OrbitVersion   string                  `json:"orbit_version"`
			PhysicalMemory int64                   `json:"physical_memory"`
			DiskAvailable  *int64                  `json:"disk_space_available_bytes,omitempty"`
			DiskTotal      *int64                  `json:"disk_space_total_bytes,omitempty"`
			DeviceMappings []contractDeviceMapping `json:"device_mappings"`
		} `json:"items"`
		Count int `json:"count"`
	}
	doJSON(t, router, http.MethodGet, "/api/hosts", nil, adminCookie, &body)
	for _, host := range body.Items {
		if host.HardwareUUID != hardwareUUID {
			continue
		}
		if host.DisplayName != "Contract Mac" {
			t.Fatalf("host display_name = %q, want Contract Mac", host.DisplayName)
		}
		if host.PhysicalMemory != 68719476736 {
			t.Fatalf("host physical_memory = %d, want 68719476736", host.PhysicalMemory)
		}
		if host.OrbitVersion != "1.47.0" {
			t.Fatalf("host orbit_version = %q, want 1.47.0", host.OrbitVersion)
		}
		if host.DiskAvailable == nil || *host.DiskAvailable != 1073741824 {
			t.Fatalf("host disk_space_available_bytes = %v, want 1073741824", host.DiskAvailable)
		}
		if host.DiskTotal == nil || *host.DiskTotal != 4294967296 {
			t.Fatalf("host disk_space_total_bytes = %v, want 4294967296", host.DiskTotal)
		}
		assertDeviceMapping(t, host.DeviceMappings, deviceEmail)
		return strconv.FormatInt(host.ID, 10)
	}
	t.Fatalf("host %q not found in admin list", hardwareUUID)
	return ""
}

type contractDeviceMapping struct {
	Email  string `json:"email"`
	Source string `json:"source"`
}

type contractSoftwareTitle struct {
	Name              string                             `json:"name"`
	Source            string                             `json:"source"`
	InstalledVersions []contractSoftwareInstalledVersion `json:"installed_versions"`
}

type contractSoftwareTitleWithCount struct {
	Name       string                    `json:"name"`
	Source     string                    `json:"source"`
	HostsCount int                       `json:"hosts_count"`
	Versions   []contractSoftwareVersion `json:"versions"`
}

type contractSoftwareVersion struct {
	Version          string `json:"version"`
	BundleIdentifier string `json:"bundle_identifier"`
}

type contractSoftwareInstalledVersion struct {
	Version              string                             `json:"version"`
	BundleIdentifier     string                             `json:"bundle_identifier"`
	InstalledPaths       []string                           `json:"installed_paths"`
	SignatureInformation []contractPathSignatureInformation `json:"signature_information"`
}

type contractPathSignatureInformation struct {
	InstalledPath    string `json:"installed_path"`
	TeamIdentifier   string `json:"team_identifier"`
	CDHashSHA256     string `json:"hash_sha256"`
	ExecutableSHA256 string `json:"executable_sha256"`
	ExecutablePath   string `json:"executable_path"`
}

func assertDeviceMapping(t *testing.T, mappings []contractDeviceMapping, email string) {
	t.Helper()
	for _, mapping := range mappings {
		if mapping.Email == email && mapping.Source == hosts.DeviceMappingSourceOrbitProfile {
			return
		}
	}
	t.Fatalf("device mapping %q with source %q not found", email, hosts.DeviceMappingSourceOrbitProfile)
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
	var body struct {
		Items []contractSoftwareTitle `json:"items"`
		Count int                     `json:"count"`
	}
	doJSON(t, router, http.MethodGet, "/api/hosts/"+hostID+"/software", nil, adminCookie, &body)
	assertExampleSoftware(t, body.Items, softwareName, bundleID)
}

func assertAdminSoftware(
	t *testing.T,
	router http.Handler,
	adminCookie *http.Cookie,
	softwareName string,
	bundleID string,
) {
	t.Helper()
	var body struct {
		Items []contractSoftwareTitleWithCount `json:"items"`
		Count int                              `json:"count"`
	}
	doJSON(t, router, http.MethodGet, "/api/software", nil, adminCookie, &body)
	if body.Count < 1 {
		t.Fatalf("software count = %d, want at least 1", body.Count)
	}
	for _, title := range body.Items {
		if title.Name == softwareName && title.Source == "apps" && title.HostsCount >= 1 &&
			hasContractVersion(title.Versions, "1.2.3", bundleID) {
			return
		}
	}
	t.Fatalf("%s title not found in admin software list", softwareName)
}

func assertExampleSoftware(t *testing.T, software []contractSoftwareTitle, softwareName string, bundleID string) {
	t.Helper()
	for _, title := range software {
		if title.Name == softwareName && title.Source == "apps" &&
			hasContractInstalledVersion(title.InstalledVersions, "1.2.3", bundleID) {
			return
		}
	}
	t.Fatalf("%s not found in host software list", softwareName)
}

func hasContractVersion(versions []contractSoftwareVersion, version string, bundleID string) bool {
	for _, got := range versions {
		if got.Version == version && got.BundleIdentifier == bundleID {
			return true
		}
	}
	return false
}

func hasContractInstalledVersion(versions []contractSoftwareInstalledVersion, version string, bundleID string) bool {
	for _, got := range versions {
		if got.Version != version || got.BundleIdentifier != bundleID || len(got.InstalledPaths) == 0 {
			continue
		}
		for _, signature := range got.SignatureInformation {
			if signature.InstalledPath == "/Applications/Example App.app" &&
				signature.TeamIdentifier == "ABCD123456" &&
				signature.CDHashSHA256 == "cdhash" &&
				signature.ExecutableSHA256 == "executable-hash" {
				return true
			}
		}
	}
	return false
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
