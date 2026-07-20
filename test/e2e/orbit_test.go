//go:build e2e

package e2e

import (
	"bytes"
	"embed"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/test/e2e/adminapi"
)

const (
	orbitFixtureHardwareUUID = "8D7A0410-6313-4EBD-A563-20EF6F2FD32C"
	orbitFixtureEmail        = "orbit.user@woodstar.test"
)

//go:embed testdata/orbit/*.json
var orbitProtocolFixtures embed.FS

type orbitFixtureEnrollResponse struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

type orbitFixtureConfigResponse struct {
	CommandLineStartupFlags json.RawMessage `json:"command_line_startup_flags"`
}

type orbitProtocolFixtureClient struct {
	t       *testing.T
	client  *http.Client
	baseURL string
}

func TestOrbit(t *testing.T) {
	const enrollSecret = "orbit-fixture-enroll-secret-0123456789abcdef"
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

	var enrolled orbitFixtureEnrollResponse
	client.postFixture("enroll.json", "/api/fleet/orbit/enroll", fixtureValues, http.StatusOK, &enrolled)
	if enrolled.OrbitNodeKey == "" {
		t.Fatal("Orbit enrollment returned an empty node key")
	}
	server.redact(enrolled.OrbitNodeKey)
	assertOrbitFixtureConfig(t, client, enrolled.OrbitNodeKey, http.StatusOK)
	assertOrbitFixtureConfig(t, client, enrolled.OrbitNodeKey, http.StatusOK)

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
		host.Agents.Osquery.Version != "5.23.1" {
		t.Fatalf("Orbit fixture host = %+v, want combined Orbit and osquery observation", host)
	}

	var reenrolled orbitFixtureEnrollResponse
	client.postFixture("enroll.json", "/api/fleet/orbit/enroll", fixtureValues, http.StatusOK, &reenrolled)
	if reenrolled.OrbitNodeKey == "" || reenrolled.OrbitNodeKey == enrolled.OrbitNodeKey {
		t.Fatal("duplicate-hardware Orbit enrollment did not rotate the node key")
	}
	server.redact(reenrolled.OrbitNodeKey)
	assertOrbitFixtureConfig(t, client, enrolled.OrbitNodeKey, http.StatusUnauthorized)
	assertOrbitFixtureConfig(t, client, reenrolled.OrbitNodeKey, http.StatusOK)
	client.request(http.MethodHead, orbitDevicePingPath(secondToken), nil, http.StatusUnauthorized, nil)
	if got := requireOnlyOrbitFixtureHost(t, server); got.Id != host.Id {
		t.Fatalf("duplicate-hardware Orbit enrollment host id = %d, want existing id %d", got.Id, host.Id)
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
) *http.Response {
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
	return response
}

func loadOrbitProtocolFixture(t *testing.T, name string, replacements map[string]any) []byte {
	t.Helper()
	return loadProtocolFixture(t, orbitProtocolFixtures, "orbit", name, replacements)
}

func assertOrbitFixtureCapabilities(t *testing.T, client orbitProtocolFixtureClient) {
	t.Helper()
	response := client.request(http.MethodHead, "/api/fleet/orbit/ping", nil, http.StatusOK, nil)
	const want = "orbit_endpoints,token_rotation,end_user_email"
	if got := response.Header.Get("X-Fleet-Capabilities"); got != want {
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
