//go:build e2e

package e2e

import (
	"bytes"
	"embed"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/test/e2e/adminapi"
)

const (
	orbitFixtureHardwareUUID = "8D7A0410-6313-4EBD-A563-20EF6F2FD32C"
	orbitFixtureEmail        = "orbit.user@woodstar.test"
	heartbeatTimeTolerance   = time.Millisecond
)

//go:embed testdata/orbit/*.json
var orbitProtocolFixtures embed.FS

type orbitFixtureEnrollResponse struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

type orbitFixtureConfigResponse struct {
	CommandLineStartupFlags json.RawMessage `json:"command_line_startup_flags"`
	ScriptExecutionTimeout  int             `json:"script_execution_timeout"`
	Notifications           struct {
		PendingScriptExecutionIDs []string `json:"pending_script_execution_ids"`
	} `json:"notifications"`
}

type orbitFixturePolicy struct {
	ID    int64  `json:"id"`
	Query string `json:"query"`
}

type orbitFixtureScriptResponse struct {
	HostID         int64  `json:"host_id"`
	ExecutionID    string `json:"execution_id"`
	ScriptContents string `json:"script_contents"`
	Timeout        int    `json:"timeout"`
}

type orbitFixturePolicyResults struct {
	Items []struct {
		Status      string `json:"status"`
		Remediation *struct {
			Status string `json:"status"`
		} `json:"remediation"`
	} `json:"items"`
}

type orbitFixtureRemediationRun struct {
	Status string `json:"status"`
	Output string `json:"output"`
}

type orbitProtocolFixtureClient struct {
	t       *testing.T
	client  *http.Client
	baseURL string
}

func TestOrbit(t *testing.T) { //nolint:cyclop,funlen // Linear protocol lifecycle; splitting would hide the order being proved.
	const enrollSecret = "orbit-fixture-enroll-secret-0123456789abcdef" //nolint:gosec // Protocol fixture secret.
	server := startTestServer(t)
	server.redact(enrollSecret)
	provisionAdmin(
		t,
		server,
		"admin@orbit.fixture.test",
		"Orbit Fixture Administrator",
		"orbit-fixture-admin-password",
	)
	createdSecret := createAgentSecret(t, server, adminapi.AgentSecretCreateAgentOrbit, enrollSecret)
	if createdSecret.Agent != "orbit" {
		t.Fatalf("created agent secret = %q, want orbit", createdSecret.Agent)
	}

	client := orbitProtocolFixtureClient{t: t, client: server.Client, baseURL: server.BaseURL}
	assertOrbitFixtureCapabilities(t, client)
	fixtureValues := map[string]any{
		"$ENROLL_SECRET": enrollSecret,
		"$HARDWARE_UUID": orbitFixtureHardwareUUID,
	}

	enrollStartedAt := time.Now()
	var enrolled orbitFixtureEnrollResponse
	client.postFixture("enroll.json", "/api/fleet/orbit/enroll", fixtureValues, http.StatusOK, &enrolled)
	enrollFinishedAt := time.Now()
	if enrolled.OrbitNodeKey == "" {
		t.Fatal("Orbit enrollment returned an empty node key")
	}
	server.redact(enrolled.OrbitNodeKey)
	enrolledHost := requireOnlyOrbitFixtureHost(t, server)
	enrolledHeartbeat := requireHeartbeat(t, enrolledHost.Heartbeats, "orbit")
	if len(enrolledHost.Heartbeats) != 1 ||
		enrolledHeartbeat.LastSeenAt.Before(enrollStartedAt.Add(-heartbeatTimeTolerance)) ||
		enrolledHeartbeat.LastSeenAt.After(enrollFinishedAt.Add(heartbeatTimeTolerance)) ||
		enrolledHost.LastContact == nil ||
		!enrolledHost.LastContact.Equal(enrolledHeartbeat.LastSeenAt) || enrolledHost.Status != "offline" ||
		enrolledHost.PublicIp != nil {
		t.Fatalf(
			"host after Orbit enrollment = %+v, heartbeat = %+v, want one bounded Orbit contact without osquery state",
			enrolledHost,
			enrolledHeartbeat,
		)
	}
	configStartedAt := time.Now()
	assertOrbitFixtureConfig(t, client, enrolled.OrbitNodeKey, http.StatusOK)
	assertOrbitFixtureConfig(t, client, enrolled.OrbitNodeKey, http.StatusOK)
	configFinishedAt := time.Now()
	configuredHost := requireOnlyOrbitFixtureHost(t, server)
	configuredHeartbeat := requireHeartbeat(t, configuredHost.Heartbeats, "orbit")
	if len(configuredHost.Heartbeats) != 1 ||
		!configuredHeartbeat.LastSeenAt.After(enrolledHeartbeat.LastSeenAt) ||
		configuredHeartbeat.LastSeenAt.Before(configStartedAt.Add(-heartbeatTimeTolerance)) ||
		configuredHeartbeat.LastSeenAt.After(configFinishedAt.Add(heartbeatTimeTolerance)) {
		t.Fatalf(
			"host after Orbit config = %+v, heartbeat = %+v, prior = %v, bounds = %v..%v, want one refreshed bounded Orbit contact",
			configuredHost,
			configuredHeartbeat,
			enrolledHeartbeat.LastSeenAt,
			configStartedAt,
			configFinishedAt,
		)
	}

	client.putFixture(
		"device_mapping.json",
		"/api/fleet/orbit/device_mapping",
		map[string]any{"$ORBIT_NODE_KEY": enrolled.OrbitNodeKey, "$EMAIL": orbitFixtureEmail},
		http.StatusOK,
		nil,
	)
	firstToken := "11111111-2222-4333-8444-555555555555"
	secondToken := "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	server.redact(firstToken, secondToken)
	setOrbitFixtureDeviceToken(t, client, enrolled.OrbitNodeKey, firstToken)
	client.request(http.MethodHead, orbitDevicePingPath(firstToken), nil, http.StatusOK, nil)
	setOrbitFixtureDeviceToken(t, client, enrolled.OrbitNodeKey, secondToken)
	client.request(http.MethodHead, orbitDevicePingPath(firstToken), nil, http.StatusUnauthorized, nil)
	client.request(http.MethodHead, orbitDevicePingPath(secondToken), nil, http.StatusOK, nil)

	var osqueryEnroll osqueryTestEnrollResponse
	client.postFixture("osquery_enroll.json", "/api/v1/osquery/enroll", fixtureValues, http.StatusOK, &osqueryEnroll)
	if osqueryEnroll.NodeKey == "" || osqueryEnroll.NodeInvalid {
		t.Fatalf(
			"osquery enrollment node key present/node_invalid = %t/%t, want true/false",
			osqueryEnroll.NodeKey != "",
			osqueryEnroll.NodeInvalid,
		)
	}
	server.redact(osqueryEnroll.NodeKey)

	var distributed osqueryTestDistributedReadResponse
	client.postFixture(
		"distributed_read.json",
		"/api/v1/osquery/distributed/read",
		map[string]any{"$OSQUERY_NODE_KEY": osqueryEnroll.NodeKey},
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
	distributedValues := orbitDistributedFixtureValues(t, osqueryEnroll.NodeKey, distributed.Queries)
	var distributedAck osqueryTestAcknowledgement
	client.postFixture(
		"distributed_write.json",
		"/api/v1/osquery/distributed/write",
		distributedValues,
		http.StatusOK,
		&distributedAck,
	)
	if distributedAck.NodeInvalid {
		t.Fatal("distributed write returned node_invalid")
	}
	var logAck osqueryTestAcknowledgement
	client.postFixture(
		"logger.json",
		"/api/v1/osquery/log",
		map[string]any{"$OSQUERY_NODE_KEY": osqueryEnroll.NodeKey},
		http.StatusOK,
		&logAck,
	)
	if logAck.NodeInvalid {
		t.Fatal("logger returned node_invalid")
	}

	host := requireOnlyOrbitFixtureHost(t, server)
	if host.Hardware.Uuid != orbitFixtureHardwareUUID ||
		host.DisplayName != "Orbit Fixture Mac" ||
		host.PrimaryUser == nil ||
		host.PrimaryUser.Email != orbitFixtureEmail ||
		host.PrimaryUser.Source != adminapi.HostPrimaryUserSourceOrbitProfile ||
		host.Agents.Orbit.Version != "1.57.0" ||
		host.Agents.Orbit.ScriptsEnabled == nil || !*host.Agents.Orbit.ScriptsEnabled ||
		host.Agents.Osquery.Version != "5.23.1" {
		t.Fatalf("Orbit fixture host = %+v, want combined Orbit and osquery observation", host)
	}
	proveOrbitPolicyRemediationLifecycle(
		t,
		server,
		client,
		enrolled.OrbitNodeKey,
		osqueryEnroll.NodeKey,
		host,
	)
	var reenrolled orbitFixtureEnrollResponse
	client.postFixture("enroll.json", "/api/fleet/orbit/enroll", fixtureValues, http.StatusOK, &reenrolled)
	if reenrolled.OrbitNodeKey == "" || reenrolled.OrbitNodeKey == enrolled.OrbitNodeKey {
		t.Fatal("duplicate-hardware Orbit enrollment did not rotate the node key")
	}
	server.redact(reenrolled.OrbitNodeKey)
	assertOrbitFixtureConfig(t, client, enrolled.OrbitNodeKey, http.StatusUnauthorized)
	assertOrbitFixtureConfig(t, client, reenrolled.OrbitNodeKey, http.StatusOK)
	client.request(http.MethodHead, orbitDevicePingPath(secondToken), nil, http.StatusUnauthorized, nil)
	var oldOsqueryConfig osqueryTestConfigResponse
	postJSON(
		t,
		client.client,
		client.baseURL+"/api/v1/osquery/config",
		osqueryTestNodeRequest{NodeKey: osqueryEnroll.NodeKey},
		&oldOsqueryConfig,
	)
	if !oldOsqueryConfig.NodeInvalid {
		t.Fatal("osquery node key from replaced host remained valid")
	}
	freshHost := requireOnlyOrbitFixtureHost(t, server)
	freshOrbitHeartbeat := requireHeartbeat(t, freshHost.Heartbeats, "orbit")
	if freshHost.Id == host.Id ||
		len(freshHost.Heartbeats) != 1 ||
		freshHost.PrimaryUser != nil ||
		freshHost.Agents.Osquery.Version != "" ||
		freshHost.LastContact == nil ||
		!freshHost.LastContact.Equal(freshOrbitHeartbeat.LastSeenAt) ||
		freshHost.Status != "offline" ||
		freshHost.PublicIp != nil {
		t.Fatalf(
			"host after Orbit re-enrollment = %+v, heartbeat = %+v, want a fresh Orbit-only host",
			freshHost,
			freshOrbitHeartbeat,
		)
	}
}

func (client orbitProtocolFixtureClient) postFixture(
	name string,
	path string,
	values map[string]any,
	wantStatus int,
	target any,
) {
	client.t.Helper()
	payload := loadOrbitProtocolFixture(client.t, name, values)
	client.request(http.MethodPost, path, payload, wantStatus, target)
}

func (client orbitProtocolFixtureClient) postJSON(
	path string,
	body any,
	wantStatus int,
	target any,
) {
	client.t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		client.t.Fatalf("encode POST %s request: %v", path, err)
	}
	client.request(http.MethodPost, path, payload, wantStatus, target)
}

func (client orbitProtocolFixtureClient) putFixture(
	name string,
	path string,
	values map[string]any,
	wantStatus int,
	target any,
) {
	client.t.Helper()
	payload := loadOrbitProtocolFixture(client.t, name, values)
	client.request(http.MethodPut, path, payload, wantStatus, target)
}

func (client orbitProtocolFixtureClient) request(
	method string,
	path string,
	payload []byte,
	wantStatus int,
	target any,
) http.Header {
	client.t.Helper()

	request, err := http.NewRequestWithContext(
		client.t.Context(),
		method,
		client.baseURL+path,
		bytes.NewReader(payload),
	)
	if err != nil {
		client.t.Fatalf("create %s %s request: %v", method, path, err)
	}
	if len(payload) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.client.Do(request)
	if err != nil {
		client.t.Fatalf("send %s %s request: %v", method, path, err)
	}
	body := readAndClose(client.t, response)
	if response.StatusCode != wantStatus {
		client.t.Fatalf(
			"%s %s status = %d, want %d: %s",
			method,
			path,
			response.StatusCode,
			wantStatus,
			strings.TrimSpace(string(body)),
		)
	}
	if target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			client.t.Fatalf("decode %s %s response: %v", method, path, err)
		}
	}
	return response.Header
}

func requestFixtureJSON(
	t *testing.T,
	client *http.Client,
	method string,
	requestURL string,
	body any,
	wantStatus int,
	target any,
) {
	t.Helper()
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("encode %s %s request: %v", method, requestURL, err)
		}
	}
	request, err := http.NewRequestWithContext(t.Context(), method, requestURL, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, requestURL, err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send %s %s request: %v", method, requestURL, err)
	}
	responseBody := readAndClose(t, response)
	if response.StatusCode != wantStatus {
		t.Fatalf(
			"%s %s status = %d, want %d: %s",
			method,
			requestURL,
			response.StatusCode,
			wantStatus,
			strings.TrimSpace(string(responseBody)),
		)
	}
	if target != nil {
		if err := json.Unmarshal(responseBody, target); err != nil {
			t.Fatalf("decode %s %s response: %v", method, requestURL, err)
		}
	}
}

func loadOrbitProtocolFixture(t *testing.T, name string, replacements map[string]any) []byte {
	t.Helper()
	return loadProtocolFixture(t, orbitProtocolFixtures, "orbit", name, replacements)
}

func assertOrbitFixtureCapabilities(t *testing.T, client orbitProtocolFixtureClient) {
	t.Helper()
	header := client.request(http.MethodHead, "/api/fleet/orbit/ping", nil, http.StatusOK, nil)
	const want = "orbit_endpoints,token_rotation,end_user_email"
	if got := header.Get("X-Fleet-Capabilities"); got != want {
		t.Fatalf("Orbit capabilities = %q, want %q", got, want)
	}
}

func assertOrbitFixtureConfig(
	t *testing.T,
	client orbitProtocolFixtureClient,
	nodeKey string,
	wantStatus int,
) {
	t.Helper()
	var response orbitFixtureConfigResponse
	target := any(nil)
	if wantStatus == http.StatusOK {
		target = &response
	}
	client.postFixture(
		"config.json",
		"/api/fleet/orbit/config",
		map[string]any{"$ORBIT_NODE_KEY": nodeKey},
		wantStatus,
		target,
	)
	if wantStatus != http.StatusOK {
		return
	}
	var flags map[string]any
	if err := json.Unmarshal(response.CommandLineStartupFlags, &flags); err != nil {
		t.Fatalf("decode Orbit startup flags: %v", err)
	}
	if flags["disable_carver"] != true ||
		flags["carver_disable_function"] != true ||
		flags["logger_min_status"] != float64(4) {
		t.Fatalf("Orbit startup flags = %+v, want Woodstar defaults", flags)
	}
	if response.ScriptExecutionTimeout != 300 {
		t.Fatalf("Orbit script timeout = %d, want 300", response.ScriptExecutionTimeout)
	}
}

func proveOrbitPolicyRemediationLifecycle( //nolint:funlen // Linear cross-protocol policy and Orbit lifecycle.
	t *testing.T,
	server *testServer,
	client orbitProtocolFixtureClient,
	orbitNodeKey string,
	osqueryNodeKey string,
	host adminapi.Host,
) {
	t.Helper()

	labelsResponse, err := server.Admin.ListLabelsWithResponse(
		t.Context(),
		&adminapi.ListLabelsParams{LabelType: new(adminapi.ListLabelsParamsLabelType("builtin"))},
	)
	labelsResponse = requireAPIResponse(t, "list policy target labels", http.StatusOK, labelsResponse, err)
	if labelsResponse.JSON200 == nil {
		t.Fatal("list policy target labels returned no JSON body")
	}
	var allHostsLabelID int64
	for _, label := range labelsResponse.JSON200.Items {
		if label.BuiltinKey != nil && *label.BuiltinKey == "all-hosts" {
			allHostsLabelID = label.Id
			break
		}
	}
	if allHostsLabelID == 0 {
		t.Fatal("all-hosts label not found")
	}

	const (
		policyQuery       = "SELECT 1 WHERE 0;"
		remediationScript = "#!/bin/zsh\necho remediated\n"
	)
	var policy orbitFixturePolicy
	requestFixtureJSON(
		t,
		server.AdminHTTP,
		http.MethodPost,
		server.BaseURL+"/api/osquery/policies",
		map[string]any{
			"name":        "Orbit remediation fixture",
			"description": "Exercises policy-owned automatic remediation.",
			"resolution":  "The fixture script should run.",
			"query":       policyQuery,
			"targets": map[string]any{
				"include": []map[string]any{{"label_id": allHostsLabelID}},
				"exclude": []map[string]any{},
			},
			"remediation": map[string]any{
				"script":    remediationScript,
				"automatic": true,
			},
		},
		http.StatusCreated,
		&policy,
	)
	if policy.ID <= 0 || policy.Query != policyQuery {
		t.Fatalf("created policy = %+v, want remediation fixture", policy)
	}

	var distributed osqueryTestDistributedReadResponse
	postJSON(
		t,
		client.client,
		client.baseURL+"/api/v1/osquery/distributed/read",
		osqueryTestNodeRequest{NodeKey: osqueryNodeKey},
		&distributed,
	)
	var policyQueryName string
	for name, query := range distributed.Queries {
		if strings.HasPrefix(name, "woodstar_policy_query_") && query == policyQuery {
			policyQueryName = name
			break
		}
	}
	if policyQueryName == "" {
		t.Fatalf("distributed policy queries = %+v, want remediation fixture", distributed.Queries)
	}

	var acknowledgement osqueryTestAcknowledgement
	postJSON(
		t,
		client.client,
		client.baseURL+"/api/v1/osquery/distributed/write",
		osqueryTestDistributedWriteRequest{
			NodeKey:  osqueryNodeKey,
			Queries:  map[string][]map[string]string{policyQueryName: {}},
			Statuses: map[string]json.RawMessage{policyQueryName: json.RawMessage(`0`)},
			Messages: map[string]string{},
		},
		&acknowledgement,
	)
	if acknowledgement.NodeInvalid {
		t.Fatal("policy result returned node_invalid")
	}

	var config orbitFixtureConfigResponse
	client.postFixture(
		"config.json",
		"/api/fleet/orbit/config",
		map[string]any{"$ORBIT_NODE_KEY": orbitNodeKey},
		http.StatusOK,
		&config,
	)
	if len(config.Notifications.PendingScriptExecutionIDs) != 1 {
		t.Fatalf("pending Orbit scripts = %+v, want one", config.Notifications.PendingScriptExecutionIDs)
	}
	executionID := config.Notifications.PendingScriptExecutionIDs[0]
	server.redact(executionID)

	requestBody := map[string]any{
		"orbit_node_key": orbitNodeKey,
		"execution_id":   executionID,
	}
	var script orbitFixtureScriptResponse
	client.postJSON("/api/fleet/orbit/scripts/request", requestBody, http.StatusOK, &script)
	if script.HostID != host.Id || script.ExecutionID != executionID ||
		script.ScriptContents != remediationScript || script.Timeout != 300 {
		t.Fatalf("claimed Orbit script = %+v, want immutable policy remediation", script)
	}
	client.postJSON("/api/fleet/orbit/scripts/request", requestBody, http.StatusNotFound, nil)
	client.postJSON(
		"/api/fleet/orbit/scripts/result",
		map[string]any{
			"orbit_node_key": orbitNodeKey,
			"execution_id":   executionID,
			"output":         "remediated\n",
			"runtime":        1,
			"exit_code":      0,
			"timeout":        300,
		},
		http.StatusOK,
		nil,
	)

	var results orbitFixturePolicyResults
	requestFixtureJSON(
		t,
		server.AdminHTTP,
		http.MethodGet,
		server.BaseURL+"/api/osquery/policies/"+strconv.FormatInt(policy.ID, 10)+"/results",
		nil,
		http.StatusOK,
		&results,
	)
	if len(results.Items) != 1 || results.Items[0].Status != "fail" ||
		results.Items[0].Remediation == nil || results.Items[0].Remediation.Status != "succeeded" {
		t.Fatalf("policy result after script success = %+v, want fail with succeeded remediation", results.Items)
	}

	var run orbitFixtureRemediationRun
	requestFixtureJSON(
		t,
		server.AdminHTTP,
		http.MethodGet,
		server.BaseURL+"/api/osquery/policies/"+strconv.FormatInt(policy.ID, 10)+
			"/hosts/"+strconv.FormatInt(host.Id, 10)+"/remediation",
		nil,
		http.StatusOK,
		&run,
	)
	if run.Status != "succeeded" || run.Output != "remediated\n" {
		t.Fatalf("latest remediation run = %+v, want succeeded output", run)
	}
}

func setOrbitFixtureDeviceToken(
	t *testing.T,
	client orbitProtocolFixtureClient,
	nodeKey string,
	token string,
) {
	t.Helper()
	client.postFixture(
		"device_token.json",
		"/api/fleet/orbit/device_token",
		map[string]any{"$ORBIT_NODE_KEY": nodeKey, "$DEVICE_AUTH_TOKEN": token},
		http.StatusOK,
		nil,
	)
}

func orbitDevicePingPath(token string) string {
	return "/api/latest/fleet/device/" + url.PathEscape(token) + "/ping"
}

func orbitDistributedFixtureValues(
	t *testing.T,
	nodeKey string,
	queries map[string]string,
) map[string]any {
	t.Helper()

	values := map[string]any{"$OSQUERY_NODE_KEY": nodeKey}
	for token, suffix := range map[string]string{
		"$QUERY_SYSTEM_INFO":  "system_info",
		"$QUERY_OS_VERSION":   "os_version",
		"$QUERY_OSQUERY_INFO": "osquery_info",
		"$QUERY_ORBIT_INFO":   "orbit_info",
	} {
		for name := range queries {
			if strings.TrimPrefix(name, "woodstar_detail_query_") == suffix {
				values[token] = name
				break
			}
		}
		if _, ok := values[token]; !ok {
			t.Fatalf("distributed work did not include required Orbit fixture query %q", suffix)
		}
	}
	return values
}

func requireOnlyOrbitFixtureHost(t *testing.T, server *testServer) adminapi.Host {
	t.Helper()
	response, err := server.Admin.ListHostsWithResponse(t.Context(), nil)
	response = requireAPIResponse(t, "list Orbit fixture hosts", http.StatusOK, response, err)
	if response.JSON200 == nil {
		t.Fatal("list Orbit fixture hosts returned no JSON body")
	}
	hosts := *response.JSON200
	if hosts.Count != 1 || len(hosts.Items) != 1 {
		t.Fatalf("Orbit fixture host count/items = %d/%d, want 1/1", hosts.Count, len(hosts.Items))
	}
	return hosts.Items[0]
}

func requireHeartbeat(t *testing.T, heartbeats []adminapi.Heartbeat, source string) adminapi.Heartbeat {
	t.Helper()

	var match adminapi.Heartbeat
	count := 0
	for _, heartbeat := range heartbeats {
		if heartbeat.Source == source {
			match = heartbeat
			count++
		}
	}
	if count != 1 {
		t.Fatalf("heartbeat source %q count = %d in %+v, want exactly 1", source, count, heartbeats)
	}
	return match
}
