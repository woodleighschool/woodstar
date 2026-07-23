//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/test/e2e/adminapi"
)

type osqueryTestEnrollRequest struct {
	EnrollSecret   string                       `json:"enroll_secret"`
	HostIdentifier string                       `json:"host_identifier"`
	HostDetails    map[string]map[string]string `json:"host_details"`
}

type osqueryTestEnrollResponse struct {
	NodeKey     string `json:"node_key"`
	NodeInvalid bool   `json:"node_invalid"`
}

type osqueryTestNodeRequest struct {
	NodeKey string `json:"node_key"`
}

type osqueryTestScheduleEntry struct {
	Query    string `json:"query"`
	Interval int32  `json:"interval"`
	Snapshot bool   `json:"snapshot"`
	Version  string `json:"version"`
}

type osqueryTestConfigResponse struct {
	NodeInvalid bool                                `json:"node_invalid"`
	Schedule    map[string]osqueryTestScheduleEntry `json:"schedule"`
	Options     map[string]string                   `json:"options"`
}

type osqueryTestDistributedReadResponse struct {
	NodeInvalid bool              `json:"node_invalid"`
	Queries     map[string]string `json:"queries"`
	Discovery   map[string]string `json:"discovery"`
}

type osqueryTestDistributedWriteRequest struct {
	NodeKey  string                         `json:"node_key"`
	Queries  map[string][]map[string]string `json:"queries"`
	Statuses map[string]json.RawMessage     `json:"statuses"`
	Messages map[string]string              `json:"messages"`
}

type osqueryTestAcknowledgement struct {
	NodeInvalid bool `json:"node_invalid"`
}

type osqueryTestLogRequest struct {
	NodeKey string          `json:"node_key"`
	LogType string          `json:"log_type"`
	Data    json.RawMessage `json:"data"`
}

func TestOsquery(t *testing.T) { //nolint:cyclop,funlen,gocognit // Linear protocol lifecycle; splitting would hide the order being proved.
	const (
		enrollSecret   = "osquery-integration-secret-0123456789abcdef"
		hostIdentifier = "osquery-integration-mac"
		hardwareUUID   = "8D7A0410-6313-4EBD-A563-20EF6F2FD32C"
		softwareName   = "Visual Studio Code"
		munkiName      = "VisualStudioCode"
		bundleID       = "com.microsoft.VSCode"
		reportUnixTime = int64(1778848496)
	)

	server := startTestServer(t)
	server.redact(enrollSecret)
	agentClient := verifyingClient(t, server.CACertificate)

	provisionAdmin(
		t,
		server,
		"admin@woodstar.test",
		"Integration Administrator",
		"integration-admin-password",
	)

	createdSecret := createAgentSecret(t, server, adminapi.AgentSecretCreateAgentOrbit, enrollSecret)
	if createdSecret.Agent != "orbit" {
		t.Fatalf("created agent = %q, want orbit", createdSecret.Agent)
	}

	labelsResponse, err := server.Admin.ListLabelsWithResponse(
		t.Context(),
		&adminapi.ListLabelsParams{LabelType: new(adminapi.ListLabelsParamsLabelType("builtin"))},
	)
	labelsResponse = requireAPIResponse(t, "list builtin labels", http.StatusOK, labelsResponse, err)
	if labelsResponse.JSON200 == nil {
		t.Fatal("list builtin labels returned no JSON body")
	}
	labels := *labelsResponse.JSON200
	var allHostsLabelID int64
	for _, label := range labels.Items {
		if label.BuiltinKey != nil && *label.BuiltinKey == "all-hosts" {
			allHostsLabelID = label.Id
			break
		}
	}
	if allHostsLabelID == 0 {
		t.Fatalf("builtin labels = %+v, want all-hosts", labels.Items)
	}

	munkiSoftwareResponse, err := server.Admin.CreateMunkiSoftwareWithResponse(
		t.Context(),
		adminapi.MunkiCreateMutation{
			Name: munkiName,
			Targets: adminapi.MunkiTargets{
				Include: []adminapi.MunkiInclude{{
					LabelId: allHostsLabelID,
					Package: adminapi.MunkiPackageSelector{Strategy: "latest"},
					Actions: []adminapi.MunkiIncludeActions{"managed_updates"},
				}},
				Exclude: []adminapi.LabelRef{},
			},
		},
	)
	munkiSoftwareResponse = requireAPIResponse(
		t,
		"create Munki software",
		http.StatusCreated,
		munkiSoftwareResponse,
		err,
	)
	if munkiSoftwareResponse.JSON201 == nil {
		t.Fatal("create Munki software returned no JSON body")
	}
	munkiSoftware := *munkiSoftwareResponse.JSON201
	nopkg := adminapi.MunkiPackageCreateMutationInstallerType("nopkg")
	munkiPackageResponse, err := server.Admin.CreateMunkiPackageWithResponse(
		t.Context(),
		adminapi.MunkiPackageCreateMutation{
			SoftwareId:    munkiSoftware.Id,
			Version:       "1.130.0",
			InstallerType: &nopkg,
			OnDemand:      new(true),
		},
	)
	munkiPackageResponse = requireAPIResponse(
		t,
		"create Munki package",
		http.StatusCreated,
		munkiPackageResponse,
		err,
	)
	if munkiPackageResponse.JSON201 == nil {
		t.Fatal("create Munki package returned no JSON body")
	}

	minOsqueryVersion := "5.12.0"
	reportMutation := adminapi.OsqueryReportMutation{
		Name:              "Installed applications",
		Description:       new("Visible integration snapshot"),
		Query:             "SELECT name, version FROM apps;",
		MinOsqueryVersion: &minOsqueryVersion,
		ScheduleInterval:  new(int32(300)),
		Targets: adminapi.OsqueryReportTargets{
			Include: []adminapi.LabelRef{{LabelId: allHostsLabelID}},
			Exclude: []adminapi.LabelRef{},
		},
	}
	reportResponse, err := server.Admin.CreateOsqueryReportWithResponse(t.Context(), reportMutation)
	reportResponse = requireAPIResponse(t, "create osquery report", http.StatusCreated, reportResponse, err)
	if reportResponse.JSON201 == nil {
		t.Fatal("create osquery report returned no JSON body")
	}
	report := *reportResponse.JSON201
	if report.Name != reportMutation.Name || report.Query != reportMutation.Query ||
		report.ScheduleInterval != *reportMutation.ScheduleInterval {
		t.Fatalf("created report = %+v, want requested scheduled report", report)
	}

	var enroll osqueryTestEnrollResponse
	postJSON(
		t,
		agentClient,
		server.BaseURL+"/api/v1/osquery/enroll",
		osqueryTestEnrollRequest{
			EnrollSecret:   enrollSecret,
			HostIdentifier: hostIdentifier,
			HostDetails: map[string]map[string]string{
				"system_info": {
					"uuid":               hardwareUUID,
					"hostname":           "osquery-mac.woodstar.test",
					"computer_name":      "Osquery Integration Mac",
					"hardware_serial":    "C02OSQUERYTEST",
					"hardware_model":     "Mac15,8",
					"hardware_vendor":    "Apple Inc.",
					"cpu_type":           "arm64",
					"cpu_subtype":        "arm64e",
					"cpu_brand":          "Apple M4",
					"cpu_logical_cores":  "10",
					"cpu_physical_cores": "10",
					"physical_memory":    "68719476736",
				},
				"osquery_info": {"version": "5.12.0"},
				"os_version": {
					"name":     "macOS",
					"version":  "26.5",
					"build":    "25F5068a",
					"platform": "darwin",
				},
				"platform_info": {"extra": "Darwin Kernel Version 25.5.0"},
			},
		},
		&enroll,
	)
	if enroll.NodeKey == "" || enroll.NodeInvalid {
		t.Fatalf(
			"enroll response node key present/node_invalid = %t/%t, want true/false",
			enroll.NodeKey != "",
			enroll.NodeInvalid,
		)
	}
	server.redact(enroll.NodeKey)

	var config osqueryTestConfigResponse
	postJSON(
		t,
		agentClient,
		server.BaseURL+"/api/v1/osquery/config",
		osqueryTestNodeRequest{NodeKey: enroll.NodeKey},
		&config,
	)
	if config.NodeInvalid {
		t.Fatal("config returned node_invalid for enrolled host")
	}
	var scheduleName string
	for name, entry := range config.Schedule {
		if entry.Query != reportMutation.Query {
			continue
		}
		if entry.Interval != *reportMutation.ScheduleInterval || !entry.Snapshot ||
			entry.Version != minOsqueryVersion {
			t.Fatalf("schedule entry = %+v, want requested interval, snapshot, and minimum version", entry)
		}
		scheduleName = name
		break
	}
	if scheduleName == "" {
		t.Fatalf("config schedule = %+v, want created report query", config.Schedule)
	}
	for name, want := range map[string]string{
		"disable_distributed": "false",
		"disable_carver":      "true",
		"logger_min_status":   "4",
	} {
		if config.Options[name] != want {
			t.Fatalf("config option %q = %q, want %q", name, config.Options[name], want)
		}
	}

	var distributed osqueryTestDistributedReadResponse
	postJSON(
		t,
		agentClient,
		server.BaseURL+"/api/v1/osquery/distributed/read",
		osqueryTestNodeRequest{NodeKey: enroll.NodeKey},
		&distributed,
	)
	if distributed.NodeInvalid || len(distributed.Queries) == 0 {
		t.Fatalf(
			"distributed read node_invalid/query count = %t/%d, want false/positive",
			distributed.NodeInvalid,
			len(distributed.Queries),
		)
	}

	queryRows := make(map[string][]map[string]string, len(distributed.Queries))
	statuses := make(map[string]json.RawMessage, len(distributed.Queries))
	requiredOverlays := make(map[string]bool)
	for name := range distributed.Queries {
		queryRows[name] = []map[string]string{}
		statuses[name] = json.RawMessage(`0`)
		suffix, ok := strings.CutPrefix(name, "woodstar_detail_query_")
		if !ok {
			continue
		}
		switch suffix {
		case "os_version":
			queryRows[name] = []map[string]string{{
				"name": "macOS", "version": "26.5", "build": "25F5068a", "platform": "darwin",
			}}
			requiredOverlays[suffix] = true
		case "system_info":
			queryRows[name] = []map[string]string{{
				"uuid": hardwareUUID, "hostname": "osquery-mac.woodstar.test",
				"computer_name": "Osquery Integration Mac", "hardware_serial": "C02OSQUERYTEST",
				"hardware_model": "Mac15,8", "hardware_vendor": "Apple Inc.",
				"cpu_type": "arm64", "cpu_subtype": "arm64e", "cpu_brand": "Apple M4",
				"cpu_logical_cores": "10", "cpu_physical_cores": "10", "physical_memory": "68719476736",
			}}
			requiredOverlays[suffix] = true
		case "osquery_info":
			queryRows[name] = []map[string]string{{"version": "5.12.0"}}
			requiredOverlays[suffix] = true
		case "osquery_flags":
			queryRows[name] = []map[string]string{
				{"name": "distributed_interval", "value": "15"},
				{"name": "config_tls_refresh", "value": "60"},
			}
		case "orbit_info":
			queryRows[name] = []map[string]string{{"version": "1.47.0"}}
		case "uptime":
			queryRows[name] = []map[string]string{{"total_seconds": "3600"}}
			requiredOverlays[suffix] = true
		case "root_disk_darwin":
			queryRows[name] = []map[string]string{{"bytes_available": "1073741824", "bytes_total": "4294967296"}}
			requiredOverlays[suffix] = true
		case "primary_interface_unix":
			queryRows[name] = []map[string]string{{"primary_ip": "192.0.2.10", "primary_mac": "aa:bb:cc:dd:ee:ff"}}
			requiredOverlays[suffix] = true
		case "users":
			queryRows[name] = []map[string]string{{
				"uid": "501", "username": "integration", "type": "local",
				"description": "Integration User", "directory": "/Users/integration", "shell": "/bin/zsh",
			}}
			requiredOverlays[suffix] = true
		case "software_macos":
			queryRows[name] = []map[string]string{{
				"name": softwareName, "version": "1.129.1", "source": "apps",
				"bundle_identifier": bundleID, "installed_path": "/Applications/Visual Studio Code.app",
			}}
			requiredOverlays[suffix] = true
		case "software_macos_codesign":
			queryRows[name] = []map[string]string{
				{
					"path":            "/Applications/Visual Studio Code.app",
					"team_identifier": "WOODSTAR01",
					"cdhash_sha256":   "cdhash",
				},
			}
		case "software_macos_executable_sha256":
			queryRows[name] = []map[string]string{{
				"path": "/Applications/Visual Studio Code.app", "executable_sha256": "executable-hash",
				"executable_path": "/Applications/Visual Studio Code.app/Contents/MacOS/Electron",
			}}
		case "munki_info":
			queryRows[name] = []map[string]string{{
				"version": "7.1.2.5700", "manifest_name": "site_default",
				"errors": "first error;second error", "warnings": "first warning",
				"problem_installs": "Broken App", "start_time": "2026-05-31 19:23:00 +1000",
				"end_time": "2026-05-31 19:24:14 +1000",
			}}
		case "munki_installs":
			queryRows[name] = []map[string]string{{
				"name": munkiName, "display_name": softwareName,
				"installed": "false", "installed_version": "", "version_to_install": "1.130.0",
			}}
		}
	}
	for _, suffix := range []string{
		"os_version", "system_info", "osquery_info", "uptime", "root_disk_darwin",
		"primary_interface_unix", "users", "software_macos",
	} {
		if !requiredOverlays[suffix] {
			t.Fatalf("distributed work did not include required detail query %q", suffix)
		}
	}

	var writeAck osqueryTestAcknowledgement
	postJSON(
		t,
		agentClient,
		server.BaseURL+"/api/v1/osquery/distributed/write",
		osqueryTestDistributedWriteRequest{
			NodeKey: enroll.NodeKey, Queries: queryRows, Statuses: statuses, Messages: map[string]string{},
		},
		&writeAck,
	)
	if writeAck.NodeInvalid {
		t.Fatal("distributed write returned node_invalid")
	}

	var secondRead osqueryTestDistributedReadResponse
	postJSON(
		t,
		agentClient,
		server.BaseURL+"/api/v1/osquery/distributed/read",
		osqueryTestNodeRequest{NodeKey: enroll.NodeKey},
		&secondRead,
	)
	if secondRead.NodeInvalid || len(secondRead.Queries) != 0 {
		t.Fatalf(
			"second distributed read node_invalid/query count = %t/%d, want false/0",
			secondRead.NodeInvalid,
			len(secondRead.Queries),
		)
	}

	var statusAck osqueryTestAcknowledgement
	postJSON(
		t,
		agentClient,
		server.BaseURL+"/api/v1/osquery/log",
		osqueryTestLogRequest{
			NodeKey: enroll.NodeKey,
			LogType: "status",
			Data:    json.RawMessage(`[{"severity":0,"filename":"init.cpp","line":1,"message":"osquery initialized"}]`),
		},
		&statusAck,
	)
	if statusAck.NodeInvalid {
		t.Fatal("status log returned node_invalid")
	}

	snapshotData, err := json.Marshal([]struct {
		Name     string              `json:"name"`
		UnixTime int64               `json:"unixTime"`
		Action   string              `json:"action"`
		Snapshot []map[string]string `json:"snapshot"`
	}{
		{
			Name: scheduleName, UnixTime: reportUnixTime, Action: "snapshot",
			Snapshot: []map[string]string{
				{"name": "Alpha", "version": "1.0"},
				{"name": "Bravo", "version": "2.0"},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode report snapshot: %v", err)
	}
	var resultAck osqueryTestAcknowledgement
	postJSON(
		t,
		agentClient,
		server.BaseURL+"/api/v1/osquery/log",
		osqueryTestLogRequest{NodeKey: enroll.NodeKey, LogType: "result", Data: snapshotData},
		&resultAck,
	)
	if resultAck.NodeInvalid {
		t.Fatal("result log returned node_invalid")
	}

	hostListResponse, err := server.Admin.ListHostsWithResponse(t.Context(), nil)
	hostListResponse = requireAPIResponse(t, "list hosts", http.StatusOK, hostListResponse, err)
	if hostListResponse.JSON200 == nil {
		t.Fatal("list hosts returned no JSON body")
	}
	hostList := *hostListResponse.JSON200
	if hostList.Count != 1 || len(hostList.Items) != 1 {
		t.Fatalf("host list count/items = %d/%d, want 1/1", hostList.Count, len(hostList.Items))
	}
	host := hostList.Items[0]
	if host.Hardware.Uuid != hardwareUUID || host.Enrollment.Agent != "osquery" ||
		host.DisplayName != "Osquery Integration Mac" || host.Status != "online" {
		t.Fatalf("host identity/enrollment = %+v, want enrolled online osquery Mac", host)
	}

	hostDetailResponse, err := server.Admin.GetHostWithResponse(t.Context(), host.Id)
	hostDetailResponse = requireAPIResponse(t, "get host", http.StatusOK, hostDetailResponse, err)
	if hostDetailResponse.JSON200 == nil {
		t.Fatal("get host returned no JSON body")
	}
	hostDetail := *hostDetailResponse.JSON200
	if hostDetail.Hostname != "osquery-mac.woodstar.test" ||
		hostDetail.ComputerName != "Osquery Integration Mac" ||
		hostDetail.Hardware.Serial != "C02OSQUERYTEST" ||
		hostDetail.Hardware.Vendor != "Apple Inc." ||
		hostDetail.Hardware.ModelIdentifier != "Mac15,8" ||
		hostDetail.Hardware.MemoryBytes != 68719476736 ||
		hostDetail.Hardware.Cpu.Architecture != "arm64" ||
		hostDetail.Hardware.Cpu.Brand != "Apple M4" ||
		hostDetail.Hardware.Cpu.PhysicalCores != 10 ||
		hostDetail.Hardware.Cpu.LogicalCores != 10 {
		t.Fatalf("host hardware/detail = %+v, want projected macOS source rows", hostDetail)
	}
	if hostDetail.Os.Platform != "darwin" || hostDetail.Os.Name != "macOS" ||
		hostDetail.Os.Version != "26.5" || hostDetail.Os.Build != "25F5068a" ||
		hostDetail.Os.KernelVersion != "Darwin Kernel Version 25.5.0" {
		t.Fatalf("host OS = %+v, want projected macOS 26.5", hostDetail.Os)
	}
	if hostDetail.Network.PrimaryIp == nil || *hostDetail.Network.PrimaryIp != "192.0.2.10" ||
		hostDetail.Network.PrimaryMac != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("host network = %+v, want projected primary interface", hostDetail.Network)
	}
	if hostDetail.Storage.BootVolume.AvailableBytes == nil ||
		*hostDetail.Storage.BootVolume.AvailableBytes != 1073741824 ||
		hostDetail.Storage.BootVolume.TotalBytes == nil ||
		*hostDetail.Storage.BootVolume.TotalBytes != 4294967296 {
		t.Fatalf("host storage = %+v, want projected root disk", hostDetail.Storage)
	}
	if hostDetail.Agents.Osquery.Version != "5.12.0" || hostDetail.Agents.Orbit.Version != "1.47.0" ||
		hostDetail.Agents.Osquery.DistributedIntervalSeconds == nil ||
		*hostDetail.Agents.Osquery.DistributedIntervalSeconds != 15 ||
		hostDetail.Agents.Osquery.ConfigRefreshSeconds == nil ||
		*hostDetail.Agents.Osquery.ConfigRefreshSeconds != 60 {
		t.Fatalf("host agents = %+v, want osquery/orbit versions and observed intervals", hostDetail.Agents)
	}
	if hostDetail.Timestamps.LastSeenAt == nil || hostDetail.Timestamps.InventoryUpdatedAt == nil ||
		hostDetail.Timestamps.LastRestartedAt == nil {
		t.Fatalf(
			"host timestamps = %+v, want last seen, fresh inventory, and restart observation",
			hostDetail.Timestamps,
		)
	}
	if len(hostDetail.Users) != 1 || hostDetail.Users[0].Uid != "501" ||
		hostDetail.Users[0].Username != "integration" || hostDetail.Users[0].Type != "local" ||
		hostDetail.Users[0].Directory != "/Users/integration" || hostDetail.Users[0].Shell != "/bin/zsh" {
		t.Fatalf("host users = %+v, want projected integration account", hostDetail.Users)
	}

	softwareResponse, err := server.Admin.ListHostSoftwareWithResponse(t.Context(), host.Id, nil)
	softwareResponse = requireAPIResponse(t, "list host software", http.StatusOK, softwareResponse, err)
	if softwareResponse.JSON200 == nil {
		t.Fatal("list host software returned no JSON body")
	}
	software := *softwareResponse.JSON200
	if software.Count != 1 || len(software.Items) != 1 || software.Items[0].Name != softwareName ||
		software.Items[0].Source != "apps" || len(software.Items[0].InstalledVersions) != 1 {
		t.Fatalf("host software = %+v, want projected integration app", software)
	}
	installed := software.Items[0].InstalledVersions[0]
	if installed.Version != "1.129.1" || installed.BundleIdentifier != bundleID ||
		len(
			installed.InstalledPaths,
		) != 1 || installed.InstalledPaths[0] != "/Applications/Visual Studio Code.app" ||
		len(installed.SignatureInformation) != 1 ||
		installed.SignatureInformation[0].TeamIdentifier != "WOODSTAR01" ||
		installed.SignatureInformation[0].HashSha256 != "cdhash" ||
		installed.SignatureInformation[0].ExecutableSha256 != "executable-hash" {
		t.Fatalf("installed software = %+v, want version, path, and signature projection", installed)
	}

	munkiResponse, err := server.Admin.GetHostMunkiStateWithResponse(t.Context(), host.Id)
	munkiResponse = requireAPIResponse(t, "get host munki state", http.StatusOK, munkiResponse, err)
	if munkiResponse.JSON200 == nil {
		t.Fatal("get host munki state returned no JSON body")
	}
	munki := *munkiResponse.JSON200
	if munki.Version != "7.1.2.5700" || munki.ManifestName != "site_default" ||
		len(munki.Errors) != 2 || munki.Errors[0] != "first error" || munki.Errors[1] != "second error" ||
		len(munki.Warnings) != 1 || munki.Warnings[0] != "first warning" ||
		len(munki.ProblemInstalls) != 1 || munki.ProblemInstalls[0] != "Broken App" {
		t.Fatalf("host Munki state = %+v, want projected osquery Munki rows", munki)
	}

	munkiSoftwareListResponse, err := server.Admin.ListHostMunkiSoftwareWithResponse(
		t.Context(),
		host.Id,
		nil,
	)
	munkiSoftwareListResponse = requireAPIResponse(
		t,
		"list host Munki software",
		http.StatusOK,
		munkiSoftwareListResponse,
		err,
	)
	if munkiSoftwareListResponse.JSON200 == nil {
		t.Fatal("list host Munki software returned no JSON body")
	}
	hostMunkiSoftware := *munkiSoftwareListResponse.JSON200
	if hostMunkiSoftware.Count != 1 || len(hostMunkiSoftware.Items) != 1 {
		t.Fatalf("host Munki software = %+v, want one resolved manifest item", hostMunkiSoftware)
	}
	vscode := hostMunkiSoftware.Items[0]
	latestPackage, err := vscode.Package.AsMunkiHostManifestLatestPackage()
	if err != nil {
		t.Fatalf("host Munki VisualStudioCode package = %+v, want latest package: %v", vscode.Package, err)
	}
	if vscode.Software.Name != munkiName ||
		len(vscode.Actions) != 1 || vscode.Actions[0] != "managed_updates" ||
		latestPackage.Strategy != adminapi.MunkiHostManifestLatestPackageStrategyLatest ||
		vscode.Observation == nil ||
		vscode.Observation.DisplayName != softwareName ||
		vscode.Observation.Installed ||
		vscode.Observation.InstalledVersion != "" ||
		vscode.Observation.TargetVersion != "1.130.0" {
		t.Fatalf("host Munki VisualStudioCode = %+v, want exact pending update observation", vscode)
	}

	resultsResponse, err := server.Admin.ListOsqueryReportResultsWithResponse(t.Context(), report.Id)
	resultsResponse = requireAPIResponse(t, "list osquery report results", http.StatusOK, resultsResponse, err)
	if resultsResponse.JSON200 == nil {
		t.Fatal("list osquery report results returned no JSON body")
	}
	results := *resultsResponse.JSON200
	if len(results) != 2 {
		t.Fatalf("report results = %+v, want two visible snapshot rows", results)
	}
	resultVersions := make(map[string]string, len(results))
	for _, result := range results {
		if result.ReportName != reportMutation.Name || result.HostName != "Osquery Integration Mac" ||
			result.LastFetched == nil || !result.LastFetched.Equal(time.Unix(reportUnixTime, 0).UTC()) {
			t.Fatalf("report result metadata = %+v, want report, host, and submitted time", result)
		}
		resultVersions[result.Columns["name"]] = result.Columns["version"]
	}
	if resultVersions["Alpha"] != "1.0" || resultVersions["Bravo"] != "2.0" {
		t.Fatalf("report result rows = %+v, want Alpha 1.0 and Bravo 2.0", resultVersions)
	}

	unknownNodeKey := "unknown-osquery-node-key"
	server.redact(unknownNodeKey)
	var unknownConfig osqueryTestConfigResponse
	postJSON(
		t,
		agentClient,
		server.BaseURL+"/api/v1/osquery/config",
		osqueryTestNodeRequest{NodeKey: unknownNodeKey},
		&unknownConfig,
	)
	if !unknownConfig.NodeInvalid {
		t.Fatal("config with unknown node key did not return node_invalid")
	}
	hostsAfterUnknownResponse, err := server.Admin.ListHostsWithResponse(t.Context(), nil)
	hostsAfterUnknownResponse = requireAPIResponse(
		t,
		"list hosts after unknown node key",
		http.StatusOK,
		hostsAfterUnknownResponse,
		err,
	)
	if hostsAfterUnknownResponse.JSON200 == nil {
		t.Fatal("list hosts after unknown node key returned no JSON body")
	}
	hostsAfterUnknown := *hostsAfterUnknownResponse.JSON200
	if hostsAfterUnknown.Count != hostList.Count || len(hostsAfterUnknown.Items) != len(hostList.Items) {
		t.Fatalf(
			"host count/items after unknown key = %d/%d, want unchanged %d/%d",
			hostsAfterUnknown.Count,
			len(hostsAfterUnknown.Items),
			hostList.Count,
			len(hostList.Items),
		)
	}
}
