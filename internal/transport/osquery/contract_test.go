package osquery

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agents/ingest"
	coreosquery "github.com/woodleighschool/woodstar/internal/agents/osquery"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/queries"
	"github.com/woodleighschool/woodstar/internal/secrets"
	"github.com/woodleighschool/woodstar/internal/software"
)

func TestOsqueryHTTPEnrollDistributedReadAndWrite(t *testing.T) {
	database, ctx := dbtest.Open(t)
	stores := newOsqueryContractStores(database)
	router := newOsqueryContractRouter(stores)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	hardwareUUID := "osquery-contract-" + suffix
	softwareName := "Example App " + suffix
	bundleID := "com.example.osquery." + suffix

	secret, err := stores.secrets.CreateOrbitEnrollSecret(ctx)
	if err != nil {
		t.Fatalf("create enroll secret: %v", err)
	}
	t.Cleanup(func() {
		cleanupOsqueryContractRows(context.Background(), t, database, hardwareUUID, secret.Value, bundleID)
	})

	nodeKey := enrollOsqueryContractHost(t, router, secret.Value, hardwareUUID)
	detailQueries := readOsqueryContractWork(t, router, nodeKey)
	assertRequiredDetailQueries(t, detailQueries)

	writeOsqueryContractDetails(t, router, nodeKey, softwareName, bundleID)

	host, err := stores.hosts.GetByOsqueryNodeKey(ctx, nodeKey)
	if err != nil {
		t.Fatalf("get host by osquery node key: %v", err)
	}
	assertProjectedHostDetails(t, host)
	assertProjectedSoftware(t, ctx, stores.software, host.ID, softwareName, bundleID)
}

func TestOsqueryHTTPReturnsNodeInvalidForUnknownNodeKey(t *testing.T) {
	database, _ := dbtest.Open(t)
	router := newOsqueryContractRouter(newOsqueryContractStores(database))

	var body coreosquery.ConfigResponse
	doOsqueryJSON(t, router, http.MethodPost, "/api/v1/osquery/config", coreosquery.ConfigRequest{
		NodeKey: "unknown-node-key",
	}, http.StatusOK, &body)
	if !body.NodeInvalid {
		t.Fatal("config with unknown node key did not return node_invalid")
	}
}

type osqueryContractStores struct {
	hosts    *hosts.HostStore
	labels   *labels.LabelStore
	secrets  *secrets.Store
	queries  *queries.QueryStore
	checks   *queries.CheckStore
	live     *queries.LiveQueryManager
	software *software.SoftwareStore
}

func newOsqueryContractStores(database *database.DB) osqueryContractStores {
	hub := queries.NewHub()
	return osqueryContractStores{
		hosts:    hosts.NewHostStore(database),
		labels:   labels.NewLabelStore(database),
		secrets:  secrets.NewStore(database),
		queries:  queries.NewQueryStore(database),
		checks:   queries.NewCheckStore(database),
		live:     queries.NewLiveQueryManager(hub, time.Minute),
		software: software.NewSoftwareStore(database),
	}
}

func newOsqueryContractRouter(stores osqueryContractStores) http.Handler {
	logger := slog.New(slog.DiscardHandler)
	router := chi.NewRouter()
	projector := ingest.NewProjector(stores.hosts, stores.software, logger.With("component", "inventory"))
	labelEvaluator := ingest.NewLabelEvaluator(stores.labels, logger.With("component", "labels"))
	RegisterRoutes(
		router,
		coreosquery.NewService(
			stores.hosts,
			projector,
			labelEvaluator,
			stores.queries,
			stores.checks,
			stores.live,
			stores.secrets,
			logger.With("component", "osquery"),
		),
		logger,
	)
	return router
}

func enrollOsqueryContractHost(t *testing.T, router http.Handler, secret string, hardwareUUID string) string {
	t.Helper()
	var body coreosquery.EnrollResponse
	doOsqueryJSON(t, router, http.MethodPost, "/api/v1/osquery/enroll", coreosquery.EnrollRequest{
		EnrollSecret:   secret,
		HostIdentifier: hardwareUUID,
		HostDetails: map[string]map[string]string{
			"system_info": {
				"uuid":               hardwareUUID,
				"hostname":           "osquery-mac",
				"computer_name":      "Osquery Mac",
				"hardware_serial":    "C02OSQUERY",
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
	}, http.StatusOK, &body)
	if body.NodeKey == "" {
		t.Fatal("enroll returned empty osquery node key")
	}
	return body.NodeKey
}

func readOsqueryContractWork(t *testing.T, router http.Handler, nodeKey string) map[string]string {
	t.Helper()
	var body coreosquery.DistributedReadResponse
	doOsqueryJSON(t, router, http.MethodPost, "/api/v1/osquery/distributed/read", coreosquery.DistributedReadRequest{
		NodeKey: nodeKey,
	}, http.StatusOK, &body)
	if body.NodeInvalid {
		t.Fatal("distributed read returned node_invalid")
	}
	return body.Queries
}

func assertRequiredDetailQueries(t *testing.T, querySQL map[string]string) {
	t.Helper()
	for _, name := range []string{
		"woodstar_detail_query_os_version",
		"woodstar_detail_query_system_info",
		"woodstar_detail_query_osquery_info",
		"woodstar_detail_query_uptime",
		"woodstar_detail_query_root_disk",
		"woodstar_detail_query_primary_interface",
		"woodstar_detail_query_users",
		"woodstar_detail_query_software_macos",
	} {
		if querySQL[name] == "" {
			t.Fatalf("missing detail query %q", name)
		}
	}
}

func writeOsqueryContractDetails(
	t *testing.T,
	router http.Handler,
	nodeKey string,
	softwareName string,
	bundleID string,
) {
	t.Helper()
	const prefix = "woodstar_detail_query_"
	queryRows := map[string][]map[string]string{
		prefix + "os_version": {{
			"name":          "macOS",
			"version":       "26.5",
			"build":         "25F5068a",
			"platform":      "darwin",
			"platform_like": "darwin",
		}},
		prefix + "system_info": {{
			"hostname":           "osquery-mac",
			"computer_name":      "Osquery Mac",
			"hardware_serial":    "C02OSQUERY",
			"hardware_model":     "Mac15,8",
			"hardware_vendor":    "Apple Inc.",
			"cpu_brand":          "Apple M4",
			"cpu_logical_cores":  "10",
			"cpu_physical_cores": "10",
			"physical_memory":    "68719476736",
		}},
		prefix + "osquery_info": {{"version": "5.22.1"}},
		prefix + "orbit_info":   {{"version": "1.47.0"}},
		prefix + "uptime":       {{"total_seconds": "3600"}},
		prefix + "root_disk":    {{"bytes_available": "1073741824", "bytes_total": "4294967296"}},
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
	statuses := make(map[string]json.RawMessage, len(queryRows))
	for name := range queryRows {
		statuses[name] = json.RawMessage(`0`)
	}

	doOsqueryJSON(t, router, http.MethodPost, "/api/v1/osquery/distributed/write", coreosquery.DistributedWriteRequest{
		NodeKey:  nodeKey,
		Queries:  queryRows,
		Statuses: statuses,
	}, http.StatusOK, nil)
}

func assertProjectedHostDetails(t *testing.T, host *hosts.Host) {
	t.Helper()
	if host.DisplayName != "Osquery Mac" {
		t.Fatalf("host display_name = %q, want Osquery Mac", host.DisplayName)
	}
	if host.PhysicalMemory != 68719476736 {
		t.Fatalf("host physical_memory = %d, want 68719476736", host.PhysicalMemory)
	}
	if host.OrbitVersion != "1.47.0" {
		t.Fatalf("host orbit_version = %q, want 1.47.0", host.OrbitVersion)
	}
	if host.DiskSpaceAvailableBytes == nil || *host.DiskSpaceAvailableBytes != 1073741824 {
		t.Fatalf("host disk_space_available_bytes = %v, want 1073741824", host.DiskSpaceAvailableBytes)
	}
	if host.DiskSpaceTotalBytes == nil || *host.DiskSpaceTotalBytes != 4294967296 {
		t.Fatalf("host disk_space_total_bytes = %v, want 4294967296", host.DiskSpaceTotalBytes)
	}
}

func assertProjectedSoftware(
	t *testing.T,
	ctx context.Context,
	softwareStore *software.SoftwareStore,
	hostID int64,
	softwareName string,
	bundleID string,
) {
	t.Helper()
	rows, _, err := softwareStore.ListForHost(ctx, hostID, software.HostSoftwareListParams{})
	if err != nil {
		t.Fatalf("list host software: %v", err)
	}
	for _, title := range rows {
		if title.Name != softwareName || title.Source != "apps" {
			continue
		}
		for _, version := range title.InstalledVersions {
			if version.Version != "1.2.3" || version.BundleIdentifier != bundleID {
				continue
			}
			for _, signature := range version.SignatureInformation {
				if signature.InstalledPath == "/Applications/Example App.app" &&
					signature.TeamIdentifier == "ABCD123456" &&
					signature.CDHashSHA256 == "cdhash" &&
					signature.ExecutableSHA256 == "executable-hash" {
					return
				}
			}
		}
	}
	t.Fatalf("projected software %q with bundle id %q not found: %#v", softwareName, bundleID, rows)
}

func cleanupOsqueryContractRows(
	ctx context.Context,
	t *testing.T,
	database *database.DB,
	hardwareUUID string,
	secretValue string,
	bundleID string,
) {
	t.Helper()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{sql: `DELETE FROM hosts WHERE hardware_uuid = $1`, args: []any{hardwareUUID}},
		{sql: `DELETE FROM secrets WHERE value = $1`, args: []any{secretValue}},
		{sql: `DELETE FROM software_titles WHERE bundle_identifier = $1`, args: []any{bundleID}},
	} {
		if _, err := database.Pool().Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatalf("cleanup osquery contract rows: %v", err)
		}
	}
}

func doOsqueryJSON(
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
	if out != nil {
		if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
			t.Fatalf("decode %s %s response: %v; body: %s", method, path, err, rec.Body.String())
		}
	}
}
