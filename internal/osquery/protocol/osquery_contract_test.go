package protocol

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

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestOsqueryHTTPEnrollDistributedReadAndWrite(t *testing.T) {
	database, ctx := dbtest.Open(t)
	stores := newOsqueryContractStores(database)
	router := newOsqueryContractRouter(stores)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	hardwareUUID := "osquery-contract-" + suffix
	softwareName := "Example App " + suffix
	bundleID := "com.example.osquery." + suffix

	secret, err := stores.agentSecrets.Create(
		ctx,
		agentauth.AgentSecretCreate{Agent: agentauth.AgentOrbit, Value: "osquery-contract-secret-value-" + suffix},
	)
	if err != nil {
		t.Fatalf("create orbit agent secret: %v", err)
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
	assertProjectedCertificates(t, ctx, stores.hosts, host.ID)
	assertProjectedMunki(t, ctx, stores.munki, host.ID)
	assertProjectedSoftware(t, ctx, stores.software, host.ID, softwareName, bundleID)
}

func TestOsqueryHTTPReturnsNodeInvalidForUnknownNodeKey(t *testing.T) {
	database, _ := dbtest.Open(t)
	router := newOsqueryContractRouter(newOsqueryContractStores(database))

	var body osquery.ConfigResponse
	doOsqueryJSON(t, router, http.MethodPost, "/api/v1/osquery/config", osquery.ConfigRequest{
		NodeKey: "unknown-node-key",
	}, http.StatusOK, &body)
	if !body.NodeInvalid {
		t.Fatal("config with unknown node key did not return node_invalid")
	}
}

func TestOsqueryHTTPConfigCarriesScheduledQueryVersion(t *testing.T) {
	database, ctx := dbtest.Open(t)
	stores := newOsqueryContractStores(database)
	router := newOsqueryContractRouter(stores)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	hardwareUUID := "osquery-schedule-" + suffix
	secret, err := stores.agentSecrets.Create(
		ctx,
		agentauth.AgentSecretCreate{Agent: agentauth.AgentOrbit, Value: "osquery-schedule-secret-value-" + suffix},
	)
	if err != nil {
		t.Fatalf("create orbit agent secret: %v", err)
	}
	allHostsID := allHostsLabelID(t, ctx, stores.labels)
	minVersion := "6.0.0"
	report, err := stores.reports.Create(ctx, reports.ReportMutation{
		Name:              "Versioned report " + suffix,
		Query:             "select 42;",
		MinOsqueryVersion: &minVersion,
		ScheduleInterval:  60,
		Targets: reports.ReportTargets{
			Include: []targeting.LabelRef{{LabelID: allHostsID}},
		},
	}, nil)
	if err != nil {
		t.Fatalf("create scheduled report: %v", err)
	}
	t.Cleanup(func() {
		if err := stores.reports.Delete(context.Background(), report.ID); err != nil {
			t.Fatalf("cleanup scheduled report: %v", err)
		}
		cleanupOsqueryContractRows(context.Background(), t, database, hardwareUUID, secret.Value, "unused-"+suffix)
	})

	nodeKey := enrollOsqueryContractHost(t, router, secret.Value, hardwareUUID)
	var body osquery.ConfigResponse
	doOsqueryJSON(t, router, http.MethodPost, "/api/v1/osquery/config", osquery.ConfigRequest{
		NodeKey: nodeKey,
	}, http.StatusOK, &body)

	for _, entry := range body.Schedule {
		if entry.Query == "select 42;" {
			if entry.Version != "6.0.0" {
				t.Fatalf("schedule entry = %+v, want version carried through", entry)
			}
			return
		}
	}
	t.Fatalf("scheduled query missing from config: %+v", body.Schedule)
}

func TestOsqueryHTTPLogStoresScheduledReportSnapshot(t *testing.T) {
	database, ctx := dbtest.Open(t)
	stores := newOsqueryContractStores(database)
	router := newOsqueryContractRouter(stores)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	hardwareUUID := "osquery-report-" + suffix
	secret, err := stores.agentSecrets.Create(
		ctx,
		agentauth.AgentSecretCreate{Agent: agentauth.AgentOrbit, Value: "osquery-report-secret-value-" + suffix},
	)
	if err != nil {
		t.Fatalf("create orbit agent secret: %v", err)
	}
	allHostsID := allHostsLabelID(t, ctx, stores.labels)
	report, err := stores.reports.Create(ctx, reports.ReportMutation{
		Name:             "Installed apps " + suffix,
		Query:            "select name, version from apps;",
		ScheduleInterval: 60,
		Targets: reports.ReportTargets{
			Include: []targeting.LabelRef{{LabelID: allHostsID}},
		},
	}, nil)
	if err != nil {
		t.Fatalf("create scheduled report: %v", err)
	}
	t.Cleanup(func() {
		if err := stores.reports.Delete(context.Background(), report.ID); err != nil {
			t.Fatalf("cleanup scheduled report: %v", err)
		}
		cleanupOsqueryContractRows(context.Background(), t, database, hardwareUUID, secret.Value, "unused-"+suffix)
	})

	nodeKey := enrollOsqueryContractHost(t, router, secret.Value, hardwareUUID)
	host, err := stores.hosts.GetByOsqueryNodeKey(ctx, nodeKey)
	if err != nil {
		t.Fatalf("get host by osquery node key: %v", err)
	}

	doOsqueryJSON(t, router, http.MethodPost, "/api/v1/osquery/log", osquery.LogRequest{
		NodeKey: nodeKey,
		LogType: "result",
		Data: json.RawMessage(`[
			{
				"name": "woodstar_report_query_` + strconv.FormatInt(report.ID, 10) + `",
				"calendarTime": "Fri May 15 12:34:56 2026 UTC",
				"action": "snapshot",
				"snapshot": [
					{"name": "Alpha", "version": "1.0"},
					{"name": "Bravo", "version": "2.0"}
				]
			}
		]`),
	}, http.StatusOK, nil)

	results, lastFetched, err := stores.reports.HostResults(ctx, host.ID, report.ID)
	if err != nil {
		t.Fatalf("host report results: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("stored result count = %d, want 2: %+v", len(results), results)
	}
	if lastFetched == nil || !lastFetched.Equal(time.Date(2026, 5, 15, 12, 34, 56, 0, time.UTC)) {
		t.Fatalf("last fetched = %v, want parsed calendar time", lastFetched)
	}
	if results[0].Columns["name"] != "Alpha" || results[1].Columns["version"] != "2.0" {
		t.Fatalf("stored results = %+v, want snapshot rows", results)
	}
}

type osqueryContractStores struct {
	hosts        *hosts.Store
	labels       *labels.Store
	agentSecrets *agentauth.Store
	reports      *reports.Store
	checks       *checks.Store
	live         *livequery.Manager
	munki        *munki.Store
	software     *inventory.Store
}

func newOsqueryContractStores(database *database.DB) osqueryContractStores {
	return osqueryContractStores{
		hosts:        hosts.NewStore(database),
		labels:       labels.NewStore(database),
		agentSecrets: agentauth.NewStore(database),
		reports:      reports.NewStore(database),
		checks:       checks.NewStore(database),
		live:         livequery.NewManager(),
		munki:        munki.NewStore(database),
		software:     inventory.NewStore(database),
	}
}

func allHostsLabelID(t *testing.T, ctx context.Context, store *labels.Store) int64 {
	t.Helper()
	rows, _, err := store.List(ctx, labels.LabelListParams{})
	if err != nil {
		t.Fatalf("list labels: %v", err)
	}
	for _, row := range rows {
		if row.BuiltinKey != nil && *row.BuiltinKey == labels.BuiltinKeyAllHosts {
			return row.ID
		}
	}
	t.Fatalf("All Hosts label not found")
	return 0
}

func newOsqueryContractRouter(stores osqueryContractStores) http.Handler {
	logger := slog.New(slog.DiscardHandler)
	router := chi.NewRouter()
	projector := ingest.NewProjector(
		stores.hosts,
		stores.software,
		logger.With("component", "inventory"),
	)
	munkiIngestor := munki.NewDetailIngestor(stores.munki)
	projector.RegisterDetailHandler(catalog.IngestMunkiInfo, munkiIngestor.IngestInfo)
	projector.RegisterDetailHandler(catalog.IngestMunkiInstalls, munkiIngestor.IngestInstalls)
	labelEvaluator := ingest.NewLabelEvaluator(stores.labels, logger.With("component", "labels"))
	RegisterOsqueryRoutes(
		router,
		osquery.NewAgentService(osquery.Dependencies{
			HostStore:          stores.hosts,
			InventoryProjector: projector,
			LabelEvaluator:     labelEvaluator,
			ReportStore:        stores.reports,
			CheckStore:         stores.checks,
			LiveQueries:        stores.live,
			SecretStore:        stores.agentSecrets,
			Logger:             logger.With("component", "osquery"),
		}),
		logger,
	)
	return router
}

func enrollOsqueryContractHost(t *testing.T, router http.Handler, secret string, hardwareUUID string) string {
	t.Helper()
	var body osquery.EnrollResponse
	doOsqueryJSON(t, router, http.MethodPost, "/api/v1/osquery/enroll", osquery.EnrollRequest{
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
				"name":    "macOS",
				"version": "26.5",
				"build":   "25F5068a",
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
	var body osquery.DistributedReadResponse
	doOsqueryJSON(t, router, http.MethodPost, "/api/v1/osquery/distributed/read", osquery.DistributedReadRequest{
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
		"woodstar_detail_query_root_disk_darwin",
		"woodstar_detail_query_primary_interface_unix",
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
			"name":    "macOS",
			"version": "26.5",
			"build":   "25F5068a",
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
		prefix + "osquery_info":     {{"version": "5.22.1"}},
		prefix + "orbit_info":       {{"version": "1.47.0"}},
		prefix + "uptime":           {{"total_seconds": "3600"}},
		prefix + "root_disk_darwin": {{"bytes_available": "1073741824", "bytes_total": "4294967296"}},
		prefix + "primary_interface_unix": {{
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
		prefix + "munki_info": {{
			"version":          "7.1.2.5700",
			"manifest_name":    "site_default",
			"success":          "true",
			"errors":           "first error; second error",
			"warnings":         "first warning",
			"problem_installs": "Broken App",
			"start_time":       "2026-05-31 19:23:00 +1000",
			"end_time":         "2026-05-31 19:24:14 +1000",
		}},
		prefix + "munki_installs": {{
			"name":              "GoogleChrome",
			"installed":         "true",
			"installed_version": "148.0",
			"end_time":          "2026-05-31 19:24:14 +1000",
		}},
		prefix + "certificates_darwin": {{
			"sha1":              "certificate-sha1",
			"common_name":       "Example Root CA",
			"subject":           "/C=AU/O=Example Org/OU=Security/CN=Example Root CA",
			"issuer":            "/C=AU/O=Example Issuer/OU=Operations/CN=Example Issuer CA",
			"ca":                "1",
			"not_valid_after":   "1777435200",
			"not_valid_before":  "1745899200",
			"path":              "/Users/contract/Library/Keychains/login.keychain-db",
			"source":            "user",
			"key_algorithm":     "rsa",
			"key_strength":      "2048",
			"key_usage":         "Certificate Sign",
			"signing_algorithm": "sha256WithRSAEncryption",
			"serial":            "01",
		}},
	}
	statuses := make(map[string]json.RawMessage, len(queryRows))
	for name := range queryRows {
		statuses[name] = json.RawMessage(`0`)
	}

	doOsqueryJSON(t, router, http.MethodPost, "/api/v1/osquery/distributed/write", osquery.DistributedWriteRequest{
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
	if host.Hardware.MemoryBytes != 68719476736 {
		t.Fatalf("host hardware.memory_bytes = %d, want 68719476736", host.Hardware.MemoryBytes)
	}
	if host.Agents.Orbit.Version != "1.47.0" {
		t.Fatalf("host agents.orbit.version = %q, want 1.47.0", host.Agents.Orbit.Version)
	}
	if host.Storage.BootVolume.AvailableBytes == nil || *host.Storage.BootVolume.AvailableBytes != 1073741824 {
		t.Fatalf(
			"host storage.boot_volume.available_bytes = %v, want 1073741824",
			host.Storage.BootVolume.AvailableBytes,
		)
	}
	if host.Storage.BootVolume.TotalBytes == nil || *host.Storage.BootVolume.TotalBytes != 4294967296 {
		t.Fatalf("host storage.boot_volume.total_bytes = %v, want 4294967296", host.Storage.BootVolume.TotalBytes)
	}
}

func assertProjectedMunki(t *testing.T, ctx context.Context, store *munki.Store, hostID int64) {
	t.Helper()
	state, err := store.LoadHostState(ctx, hostID)
	if err != nil {
		t.Fatalf("load munki state: %v", err)
	}
	if state == nil {
		t.Fatal("munki state is nil")
	}
	if state.Version != "7.1.2.5700" || state.ManifestName != "site_default" {
		t.Fatalf("munki state = %+v, want version and manifest", state)
	}
	if state.Success == nil || !*state.Success {
		t.Fatalf("munki success = %v, want true", state.Success)
	}
	if len(state.Errors) != 2 || len(state.Warnings) != 1 || len(state.ProblemInstalls) != 1 {
		t.Fatalf(
			"munki problems = errors %#v warnings %#v installs %#v",
			state.Errors,
			state.Warnings,
			state.ProblemInstalls,
		)
	}
	if len(state.Items) != 1 || state.Items[0].Name != "GoogleChrome" ||
		!state.Items[0].Installed || state.Items[0].InstalledVersion != "148.0" {
		t.Fatalf("munki items = %+v, want GoogleChrome installed", state.Items)
	}
}

func assertProjectedCertificates(t *testing.T, ctx context.Context, hostStore *hosts.Store, hostID int64) {
	t.Helper()
	certificates, err := hostStore.ListCertificates(ctx, hostID)
	if err != nil {
		t.Fatalf("list host certificates: %v", err)
	}
	if len(certificates) != 1 {
		t.Fatalf("certificate count = %d, want 1: %#v", len(certificates), certificates)
	}
	certificate := certificates[0]
	if certificate.SHA1 != "certificate-sha1" ||
		certificate.CommonName != "Example Root CA" ||
		certificate.Subject.CommonName != "Example Root CA" ||
		certificate.Subject.Organization != "Example Org" ||
		certificate.Issuer.CommonName != "Example Issuer CA" ||
		certificate.Source != "user" ||
		certificate.Username != "contract" ||
		!certificate.CertificateAuthority {
		t.Fatalf("certificate = %#v, want projected darwin user certificate", certificate)
	}
}

func assertProjectedSoftware(
	t *testing.T,
	ctx context.Context,
	softwareStore *inventory.Store,
	hostID int64,
	softwareName string,
	bundleID string,
) {
	t.Helper()
	rows, _, err := softwareStore.ListForHost(ctx, hostID, inventory.HostSoftwareListParams{})
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
		{sql: `DELETE FROM agent_secrets WHERE value = $1`, args: []any{secretValue}},
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
