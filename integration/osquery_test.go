package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

type osqueryTestLabelList struct {
	Items []struct {
		ID         int64  `json:"id"`
		BuiltinKey string `json:"builtin_key"`
	} `json:"items"`
	Count int `json:"count"`
}

type osqueryTestLabelRef struct {
	LabelID int64 `json:"label_id"`
}

type osqueryTestReportMutation struct {
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	Query             string  `json:"query"`
	MinOsqueryVersion *string `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int32   `json:"schedule_interval"`
	Targets           struct {
		Include []osqueryTestLabelRef `json:"include"`
		Exclude []osqueryTestLabelRef `json:"exclude"`
	} `json:"targets"`
}

type osqueryTestReport struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Query            string `json:"query"`
	ScheduleInterval int32  `json:"schedule_interval"`
}

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

type osqueryTestHost struct {
	ID           int64  `json:"id"`
	DisplayName  string `json:"display_name"`
	Status       string `json:"status"`
	Hostname     string `json:"hostname"`
	ComputerName string `json:"computer_name"`
	Enrollment   struct {
		Agent string `json:"agent"`
	} `json:"enrollment"`
	Hardware struct {
		UUID            string `json:"uuid"`
		Serial          string `json:"serial"`
		Vendor          string `json:"vendor"`
		ModelIdentifier string `json:"model_identifier"`
		MemoryBytes     int64  `json:"memory_bytes"`
		CPU             struct {
			Architecture  string `json:"architecture"`
			Brand         string `json:"brand"`
			PhysicalCores int32  `json:"physical_cores"`
			LogicalCores  int32  `json:"logical_cores"`
		} `json:"cpu"`
	} `json:"hardware"`
	OS struct {
		Platform      string `json:"platform"`
		Name          string `json:"name"`
		Version       string `json:"version"`
		Build         string `json:"build"`
		KernelVersion string `json:"kernel_version"`
	} `json:"os"`
	Storage struct {
		BootVolume struct {
			AvailableBytes *int64 `json:"available_bytes"`
			TotalBytes     *int64 `json:"total_bytes"`
		} `json:"boot_volume"`
	} `json:"storage"`
	Network struct {
		PrimaryIP  *string `json:"primary_ip"`
		PrimaryMAC string  `json:"primary_mac"`
	} `json:"network"`
	Agents struct {
		Osquery struct {
			Version                    string `json:"version"`
			DistributedIntervalSeconds *int32 `json:"distributed_interval_seconds"`
			ConfigRefreshSeconds       *int32 `json:"config_refresh_seconds"`
		} `json:"osquery"`
		Orbit struct {
			Version string `json:"version"`
		} `json:"orbit"`
	} `json:"agents"`
	Timestamps struct {
		LastSeenAt         *time.Time `json:"last_seen_at"`
		InventoryUpdatedAt *time.Time `json:"inventory_updated_at"`
		LastRestartedAt    *time.Time `json:"last_restarted_at"`
	} `json:"timestamps"`
}

type osqueryTestHostList struct {
	Items []osqueryTestHost `json:"items"`
	Count int               `json:"count"`
}

type osqueryTestHostDetail struct {
	osqueryTestHost

	Users []struct {
		UID         string `json:"uid"`
		Username    string `json:"username"`
		Type        string `json:"type"`
		Description string `json:"description"`
		Directory   string `json:"directory"`
		Shell       string `json:"shell"`
	} `json:"users"`
}

type osqueryTestSoftwareList struct {
	Items []struct {
		Name              string `json:"name"`
		Source            string `json:"source"`
		InstalledVersions []struct {
			Version          string   `json:"version"`
			BundleIdentifier string   `json:"bundle_identifier"`
			InstalledPaths   []string `json:"installed_paths"`
			Signatures       []struct {
				InstalledPath    string `json:"installed_path"`
				TeamIdentifier   string `json:"team_identifier"`
				CDHashSHA256     string `json:"hash_sha256"`
				ExecutableSHA256 string `json:"executable_sha256"`
			} `json:"signature_information"`
		} `json:"installed_versions"`
	} `json:"items"`
	Count int `json:"count"`
}

type osqueryTestMunkiState struct {
	Version         string   `json:"version"`
	ManifestName    string   `json:"manifest_name"`
	Errors          []string `json:"errors"`
	Warnings        []string `json:"warnings"`
	ProblemInstalls []string `json:"problem_installs"`
	Items           []struct {
		Name             string `json:"name"`
		Installed        bool   `json:"installed"`
		InstalledVersion string `json:"installed_version"`
	} `json:"items"`
}

type osqueryTestReportResult struct {
	ReportName  string            `json:"report_name"`
	HostName    string            `json:"host_name"`
	Columns     map[string]string `json:"columns"`
	LastFetched time.Time         `json:"last_fetched"`
}

func TestOsquery(t *testing.T) {
	const (
		enrollSecret   = "osquery-integration-secret-0123456789abcdef"
		hostIdentifier = "osquery-integration-mac"
		hardwareUUID   = "8D7A0410-6313-4EBD-A563-20EF6F2FD32C"
		softwareName   = "Woodstar Integration App"
		bundleID       = "school.woodleigh.woodstar-integration"
		reportUnixTime = int64(1778848496)
	)

	server := startTestServer(t)
	server.redact(enrollSecret)
	agentClient := verifyingClient(t, server.CACertificate)

	var setupUser struct {
		Email string `json:"email"`
	}
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/setup",
		struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Password string `json:"password"`
		}{
			Email:    "admin@woodstar.test",
			Name:     "Integration Administrator",
			Password: "integration-admin-password",
		},
		http.StatusCreated,
		&setupUser,
	)
	if setupUser.Email != "admin@woodstar.test" {
		t.Fatalf("setup email = %q, want admin@woodstar.test", setupUser.Email)
	}

	var createdSecret struct {
		Agent string `json:"agent"`
	}
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/agent-secrets",
		struct {
			Agent string `json:"agent"`
			Value string `json:"value"`
		}{Agent: "orbit", Value: enrollSecret},
		http.StatusCreated,
		&createdSecret,
	)
	if createdSecret.Agent != "orbit" {
		t.Fatalf("created agent = %q, want orbit", createdSecret.Agent)
	}

	var labels osqueryTestLabelList
	requestJSON(
		t,
		server.Client,
		http.MethodGet,
		server.BaseURL+"/api/labels?label_type=builtin",
		nil,
		http.StatusOK,
		&labels,
	)
	var allHostsLabelID int64
	for _, label := range labels.Items {
		if label.BuiltinKey == "all-hosts" {
			allHostsLabelID = label.ID
			break
		}
	}
	if allHostsLabelID == 0 {
		t.Fatalf("builtin labels = %+v, want all-hosts", labels.Items)
	}

	minOsqueryVersion := "5.12.0"
	reportMutation := osqueryTestReportMutation{
		Name:              "Installed applications",
		Description:       "Visible integration snapshot",
		Query:             "SELECT name, version FROM apps;",
		MinOsqueryVersion: &minOsqueryVersion,
		ScheduleInterval:  300,
	}
	reportMutation.Targets.Include = []osqueryTestLabelRef{{LabelID: allHostsLabelID}}
	reportMutation.Targets.Exclude = []osqueryTestLabelRef{}
	var report osqueryTestReport
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/osquery/reports",
		reportMutation,
		http.StatusCreated,
		&report,
	)
	if report.Name != reportMutation.Name || report.Query != reportMutation.Query ||
		report.ScheduleInterval != reportMutation.ScheduleInterval {
		t.Fatalf("created report = %+v, want requested scheduled report", report)
	}

	var enroll osqueryTestEnrollResponse
	requestJSON(
		t,
		agentClient,
		http.MethodPost,
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
		http.StatusOK,
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
	requestJSON(
		t,
		agentClient,
		http.MethodPost,
		server.BaseURL+"/api/v1/osquery/config",
		osqueryTestNodeRequest{NodeKey: enroll.NodeKey},
		http.StatusOK,
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
		if entry.Interval != reportMutation.ScheduleInterval || !entry.Snapshot ||
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
	requestJSON(
		t,
		agentClient,
		http.MethodPost,
		server.BaseURL+"/api/v1/osquery/distributed/read",
		osqueryTestNodeRequest{NodeKey: enroll.NodeKey},
		http.StatusOK,
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
				"name": softwareName, "version": "1.2.3", "source": "apps",
				"bundle_identifier": bundleID, "installed_path": "/Applications/Woodstar Integration App.app",
			}}
			requiredOverlays[suffix] = true
		case "software_macos_codesign":
			queryRows[name] = []map[string]string{
				{
					"path":            "/Applications/Woodstar Integration App.app",
					"team_identifier": "WOODSTAR01",
					"cdhash_sha256":   "cdhash",
				},
			}
		case "software_macos_executable_sha256":
			queryRows[name] = []map[string]string{{
				"path": "/Applications/Woodstar Integration App.app", "executable_sha256": "executable-hash",
				"executable_path": "/Applications/Woodstar Integration App.app/Contents/MacOS/WoodstarIntegrationApp",
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
				"name": "GoogleChrome", "installed": "true", "installed_version": "148.0",
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
	requestJSON(
		t,
		agentClient,
		http.MethodPost,
		server.BaseURL+"/api/v1/osquery/distributed/write",
		osqueryTestDistributedWriteRequest{
			NodeKey: enroll.NodeKey, Queries: queryRows, Statuses: statuses, Messages: map[string]string{},
		},
		http.StatusOK,
		&writeAck,
	)
	if writeAck.NodeInvalid {
		t.Fatal("distributed write returned node_invalid")
	}

	var secondRead osqueryTestDistributedReadResponse
	requestJSON(
		t,
		agentClient,
		http.MethodPost,
		server.BaseURL+"/api/v1/osquery/distributed/read",
		osqueryTestNodeRequest{NodeKey: enroll.NodeKey},
		http.StatusOK,
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
	requestJSON(
		t,
		agentClient,
		http.MethodPost,
		server.BaseURL+"/api/v1/osquery/log",
		osqueryTestLogRequest{
			NodeKey: enroll.NodeKey,
			LogType: "status",
			Data:    json.RawMessage(`[{"severity":0,"filename":"init.cpp","line":1,"message":"osquery initialized"}]`),
		},
		http.StatusOK,
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
	requestJSON(
		t,
		agentClient,
		http.MethodPost,
		server.BaseURL+"/api/v1/osquery/log",
		osqueryTestLogRequest{NodeKey: enroll.NodeKey, LogType: "result", Data: snapshotData},
		http.StatusOK,
		&resultAck,
	)
	if resultAck.NodeInvalid {
		t.Fatal("result log returned node_invalid")
	}

	var hostList osqueryTestHostList
	requestJSON(
		t,
		server.Client,
		http.MethodGet,
		server.BaseURL+"/api/hosts",
		nil,
		http.StatusOK,
		&hostList,
	)
	if hostList.Count != 1 || len(hostList.Items) != 1 {
		t.Fatalf("host list count/items = %d/%d, want 1/1", hostList.Count, len(hostList.Items))
	}
	host := hostList.Items[0]
	if host.Hardware.UUID != hardwareUUID || host.Enrollment.Agent != "osquery" ||
		host.DisplayName != "Osquery Integration Mac" || host.Status != "online" {
		t.Fatalf("host identity/enrollment = %+v, want enrolled online osquery Mac", host)
	}

	var hostDetail osqueryTestHostDetail
	requestJSON(
		t,
		server.Client,
		http.MethodGet,
		fmt.Sprintf("%s/api/hosts/%d", server.BaseURL, host.ID),
		nil,
		http.StatusOK,
		&hostDetail,
	)
	if hostDetail.Hostname != "osquery-mac.woodstar.test" ||
		hostDetail.ComputerName != "Osquery Integration Mac" ||
		hostDetail.Hardware.Serial != "C02OSQUERYTEST" ||
		hostDetail.Hardware.Vendor != "Apple Inc." ||
		hostDetail.Hardware.ModelIdentifier != "Mac15,8" ||
		hostDetail.Hardware.MemoryBytes != 68719476736 ||
		hostDetail.Hardware.CPU.Architecture != "arm64" ||
		hostDetail.Hardware.CPU.Brand != "Apple M4" ||
		hostDetail.Hardware.CPU.PhysicalCores != 10 ||
		hostDetail.Hardware.CPU.LogicalCores != 10 {
		t.Fatalf("host hardware/detail = %+v, want projected macOS source rows", hostDetail.osqueryTestHost)
	}
	if hostDetail.OS.Platform != "darwin" || hostDetail.OS.Name != "macOS" ||
		hostDetail.OS.Version != "26.5" || hostDetail.OS.Build != "25F5068a" ||
		hostDetail.OS.KernelVersion != "Darwin Kernel Version 25.5.0" {
		t.Fatalf("host OS = %+v, want projected macOS 26.5", hostDetail.OS)
	}
	if hostDetail.Network.PrimaryIP == nil || *hostDetail.Network.PrimaryIP != "192.0.2.10" ||
		hostDetail.Network.PrimaryMAC != "aa:bb:cc:dd:ee:ff" {
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
	if len(hostDetail.Users) != 1 || hostDetail.Users[0].UID != "501" ||
		hostDetail.Users[0].Username != "integration" || hostDetail.Users[0].Type != "local" ||
		hostDetail.Users[0].Directory != "/Users/integration" || hostDetail.Users[0].Shell != "/bin/zsh" {
		t.Fatalf("host users = %+v, want projected integration account", hostDetail.Users)
	}

	var software osqueryTestSoftwareList
	requestJSON(
		t,
		server.Client,
		http.MethodGet,
		fmt.Sprintf("%s/api/hosts/%d/software", server.BaseURL, host.ID),
		nil,
		http.StatusOK,
		&software,
	)
	if software.Count != 1 || len(software.Items) != 1 || software.Items[0].Name != softwareName ||
		software.Items[0].Source != "apps" || len(software.Items[0].InstalledVersions) != 1 {
		t.Fatalf("host software = %+v, want projected integration app", software)
	}
	installed := software.Items[0].InstalledVersions[0]
	if installed.Version != "1.2.3" || installed.BundleIdentifier != bundleID ||
		len(
			installed.InstalledPaths,
		) != 1 || installed.InstalledPaths[0] != "/Applications/Woodstar Integration App.app" ||
		len(installed.Signatures) != 1 || installed.Signatures[0].TeamIdentifier != "WOODSTAR01" ||
		installed.Signatures[0].CDHashSHA256 != "cdhash" ||
		installed.Signatures[0].ExecutableSHA256 != "executable-hash" {
		t.Fatalf("installed software = %+v, want version, path, and signature projection", installed)
	}

	var munki osqueryTestMunkiState
	requestJSON(
		t,
		server.Client,
		http.MethodGet,
		fmt.Sprintf("%s/api/hosts/%d/munki", server.BaseURL, host.ID),
		nil,
		http.StatusOK,
		&munki,
	)
	if munki.Version != "7.1.2.5700" || munki.ManifestName != "site_default" ||
		len(munki.Errors) != 2 || munki.Errors[0] != "first error" || munki.Errors[1] != "second error" ||
		len(munki.Warnings) != 1 || munki.Warnings[0] != "first warning" ||
		len(munki.ProblemInstalls) != 1 || munki.ProblemInstalls[0] != "Broken App" ||
		len(munki.Items) != 1 || munki.Items[0].Name != "GoogleChrome" || !munki.Items[0].Installed ||
		munki.Items[0].InstalledVersion != "148.0" {
		t.Fatalf("host Munki state = %+v, want projected osquery Munki rows", munki)
	}

	var results []osqueryTestReportResult
	requestJSON(
		t,
		server.Client,
		http.MethodGet,
		fmt.Sprintf("%s/api/osquery/reports/%d/results", server.BaseURL, report.ID),
		nil,
		http.StatusOK,
		&results,
	)
	if len(results) != 2 {
		t.Fatalf("report results = %+v, want two visible snapshot rows", results)
	}
	resultVersions := make(map[string]string, len(results))
	for _, result := range results {
		if result.ReportName != reportMutation.Name || result.HostName != "Osquery Integration Mac" ||
			!result.LastFetched.Equal(time.Unix(reportUnixTime, 0).UTC()) {
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
	requestJSON(
		t,
		agentClient,
		http.MethodPost,
		server.BaseURL+"/api/v1/osquery/config",
		osqueryTestNodeRequest{NodeKey: unknownNodeKey},
		http.StatusOK,
		&unknownConfig,
	)
	if !unknownConfig.NodeInvalid {
		t.Fatal("config with unknown node key did not return node_invalid")
	}
	var hostsAfterUnknown osqueryTestHostList
	requestJSON(
		t,
		server.Client,
		http.MethodGet,
		server.BaseURL+"/api/hosts",
		nil,
		http.StatusOK,
		&hostsAfterUnknown,
	)
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
