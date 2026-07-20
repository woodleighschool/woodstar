//go:build e2e

package e2e

import (
	"bytes"
	"compress/gzip"
	"context"
	"embed"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/woodleighschool/woodstar/test/e2e/adminapi"
)

const santaProtobufContentType = "application/x-protobuf"

//go:embed testdata/santa/*.json
var santaProtocolFixtures embed.FS

type santaSyncCheckpoint struct {
	RulesReceived       int32
	RulesProcessed      int32
	ConfirmedRulesHash  string
	PendingFullSync     bool
	PendingPayloadCount int32
	PendingPreflightAt  *time.Time
	LastSyncAttemptAt   *time.Time
	LastSyncSuccessAt   *time.Time
	LastCleanSyncAt     *time.Time
	DesiredTargetCount  int32
	AppliedTargetCount  int32
}

type santaProtocolFixtureClient struct {
	t         *testing.T
	client    *http.Client
	baseURL   string
	machineID string
	secret    string
}

func TestSanta(t *testing.T) {
	const (
		machineID        = "11111111-2222-4333-8444-555555555555"
		unknownMachineID = "99999999-8888-4777-8666-555555555555"
		serial           = "SANTA-INTEGRATION-001"
		santaSecret      = "santa-integration-agent-secret-0001"
		acknowledgedHash = "11111111111111111111111111111111"
		ruleIdentifier   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		eventSHA256      = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
		eventCDHash      = "cccccccccccccccccccccccccccccccccccccccc"
		eventPath        = "/Applications/Woodstar Santa Test.app/Contents/MacOS/Woodstar Santa Test"
		ruleMessage      = "Blocked by Woodstar integration"
		ruleURL          = "https://woodstar.example.test/santa"
	)

	server := startTestServer(t)
	server.redact(santaSecret)

	provisionAdmin(
		t,
		server,
		"admin@santa.integration.test",
		"Santa Integration Admin",
		"santa-integration-password",
	)

	database, err := pgx.Connect(t.Context(), server.DatabaseURL)
	if err != nil {
		t.Fatalf("connect to isolated Woodstar database: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), databaseOperationTimeout)
		defer cancel()
		if closeErr := database.Close(ctx); closeErr != nil {
			t.Errorf("close isolated Woodstar database connection: %v", closeErr)
		}
	})

	var hostID int64
	err = database.QueryRow(t.Context(), `
INSERT INTO hosts (
    hardware_uuid,
    display_name,
    hostname,
    computer_name,
    hardware_serial,
    hardware_model_identifier,
    os_platform,
    enrollment_agent,
    enrolled_at,
    last_seen_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())
RETURNING id`,
		machineID,
		"Santa Integration Mac",
		"santa-seeded.test",
		"Santa Integration Mac",
		serial,
		"Mac15,7",
		"darwin",
		"orbit",
	).Scan(&hostID)
	if err != nil {
		t.Fatalf("seed canonical Santa host: %v", err)
	}

	labelResponse, err := server.Admin.CreateLabelWithResponse(
		t.Context(),
		adminapi.LabelMutation{
			Name:                "Santa Integration Hosts",
			LabelMembershipType: new(adminapi.LabelMutationLabelMembershipType("manual")),
			HostIds:             new([]int64{hostID}),
		},
	)
	labelResponse = requireAPIResponse(t, "create label", http.StatusCreated, labelResponse, err)
	if labelResponse.JSON201 == nil {
		t.Fatal("create label returned no JSON body")
	}
	label := *labelResponse.JSON201
	if label.Id <= 0 || label.Name != "Santa Integration Hosts" ||
		label.LabelMembershipType != "manual" || label.HostIds == nil ||
		!slices.Equal(*label.HostIds, []int64{hostID}) {
		t.Fatalf("created label = %+v, want manual Santa integration label for host %d", label, hostID)
	}

	createdSecret := createAgentSecret(t, server, adminapi.AgentSecretCreateAgentSanta, santaSecret)
	if createdSecret.Id <= 0 || createdSecret.Agent != "santa" {
		t.Fatalf("created agent secret = %+v, want active Santa secret", createdSecret)
	}

	configurationResponse, err := server.Admin.CreateSantaConfigurationWithResponse(
		t.Context(),
		adminapi.SantaConfigurationMutation{
			Name:                      "Santa Integration Configuration",
			ClientMode:                "lockdown",
			EnableBundles:             true,
			EnableTransitiveRules:     true,
			EnableAllEventUpload:      true,
			DisableUnknownEventUpload: true,
			OverrideFileAccessAction:  "audit_only",
			FullSyncIntervalSeconds:   600,
			BatchSize:                 50,
			AllowedPathRegex:          new(`^/Applications/`),
			BlockedPathRegex:          new(`^/tmp/`),
			EventDetailUrl:            new(ruleURL),
			EventDetailText:           new("More information"),
			Targets: adminapi.SantaConfigurationTargets{
				Include: []adminapi.LabelRef{{LabelId: label.Id}},
				Exclude: []adminapi.LabelRef{},
			},
		},
	)
	configurationResponse = requireAPIResponse(
		t,
		"create Santa configuration",
		http.StatusCreated,
		configurationResponse,
		err,
	)
	if configurationResponse.JSON201 == nil {
		t.Fatal("create Santa configuration returned no JSON body")
	}
	configuration := *configurationResponse.JSON201
	if configuration.Id <= 0 || configuration.Name != "Santa Integration Configuration" {
		t.Fatalf("created configuration = %+v, want Santa integration configuration", configuration)
	}

	ruleResponse, err := server.Admin.CreateSantaRuleWithResponse(
		t.Context(),
		adminapi.SantaRuleMutation{
			RuleType:      "binary",
			Identifier:    ruleIdentifier,
			Name:          "Santa Integration Rule",
			CustomMessage: new(ruleMessage),
			CustomUrl:     new(ruleURL),
			Targets: adminapi.SantaRuleTargets{
				Include: []adminapi.SantaRuleInclude{{Policy: "blocklist", LabelId: label.Id}},
				Exclude: []adminapi.LabelRef{},
			},
		},
	)
	ruleResponse = requireAPIResponse(t, "create Santa rule", http.StatusCreated, ruleResponse, err)
	if ruleResponse.JSON201 == nil {
		t.Fatal("create Santa rule returned no JSON body")
	}
	createdRule := *ruleResponse.JSON201
	if createdRule.Id <= 0 || createdRule.Identifier != ruleIdentifier {
		t.Fatalf("created rule = %+v, want public Santa integration rule", createdRule)
	}

	hostsBeforeResponse, err := server.Admin.ListHostsWithResponse(
		t.Context(),
		&adminapi.ListHostsParams{PerPage: new(int32(1000))},
	)
	hostsBeforeResponse = requireAPIResponse(t, "list hosts", http.StatusOK, hostsBeforeResponse, err)
	if hostsBeforeResponse.JSON200 == nil {
		t.Fatal("list hosts returned no JSON body")
	}
	hostsBefore := *hostsBeforeResponse.JSON200
	requireOnlySantaHost(t, hostsBefore, hostID, machineID, serial)

	client := santaProtocolFixtureClient{
		t:         t,
		client:    server.Client,
		baseURL:   server.BaseURL,
		machineID: machineID,
		secret:    santaSecret,
	}

	var firstPreflight syncv1.PreflightResponse
	client.postFixture("preflight", "preflight_clean.json", nil, &syncv1.PreflightRequest{}, &firstPreflight)
	requireSantaPreflightConfiguration(t, &firstPreflight)

	eventTime := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	var eventUpload syncv1.EventUploadResponse
	client.postFixture("eventupload", "event_upload.json", nil, &syncv1.EventUploadRequest{}, &eventUpload)
	if len(eventUpload.GetEventUploadBundleBinaries()) != 0 {
		t.Fatalf("bundle requests = %v, want none", eventUpload.GetEventUploadBundleBinaries())
	}

	firstRules := client.downloadRules()
	if len(firstRules) != 1 {
		t.Fatalf("first sync downloaded %d rules, want 1", len(firstRules))
	}
	firstRule := firstRules[0]
	if firstRule.GetIdentifier() != ruleIdentifier ||
		firstRule.GetRuleType() != syncv1.RuleType_BINARY ||
		firstRule.GetPolicy() != syncv1.Policy_BLOCKLIST ||
		firstRule.GetCustomMsg() != ruleMessage ||
		firstRule.GetCustomUrl() != ruleURL {
		t.Fatalf("first downloaded rule = %+v, want public blocklist rule", firstRule)
	}

	client.postFixture(
		"postflight",
		"postflight_clean.json",
		nil,
		&syncv1.PostflightRequest{},
		&syncv1.PostflightResponse{},
	)

	firstCheckpoint := loadSantaSyncCheckpoint(t, database, hostID)
	if firstCheckpoint.RulesReceived != 1 || firstCheckpoint.RulesProcessed != 1 ||
		firstCheckpoint.ConfirmedRulesHash != acknowledgedHash || firstCheckpoint.PendingFullSync ||
		firstCheckpoint.PendingPayloadCount != 0 || firstCheckpoint.PendingPreflightAt != nil ||
		firstCheckpoint.DesiredTargetCount != 1 || firstCheckpoint.AppliedTargetCount != 1 ||
		firstCheckpoint.LastSyncAttemptAt == nil || firstCheckpoint.LastSyncSuccessAt == nil ||
		firstCheckpoint.LastCleanSyncAt == nil {
		t.Fatalf("first sync checkpoint = %+v, want promoted clean acknowledgement", firstCheckpoint)
	}

	var secondPreflight syncv1.PreflightResponse
	client.postFixture("preflight", "preflight_normal.json", nil, &syncv1.PreflightRequest{}, &secondPreflight)
	if secondPreflight.SyncType == nil || secondPreflight.GetSyncType() != syncv1.SyncType_NORMAL {
		t.Fatalf("second sync type = %v, want present NORMAL", secondPreflight.SyncType)
	}
	secondRules := client.downloadRules()
	if len(secondRules) != 0 {
		t.Fatalf("second sync downloaded rules = %+v, want none", secondRules)
	}
	client.postFixture(
		"postflight",
		"postflight_normal.json",
		nil,
		&syncv1.PostflightRequest{},
		&syncv1.PostflightResponse{},
	)

	secondCheckpoint := loadSantaSyncCheckpoint(t, database, hostID)
	if secondCheckpoint.RulesReceived != 0 || secondCheckpoint.RulesProcessed != 0 ||
		secondCheckpoint.ConfirmedRulesHash != acknowledgedHash || secondCheckpoint.PendingFullSync ||
		secondCheckpoint.PendingPayloadCount != 0 || secondCheckpoint.PendingPreflightAt != nil ||
		secondCheckpoint.DesiredTargetCount != 1 || secondCheckpoint.AppliedTargetCount != 1 ||
		secondCheckpoint.LastSyncAttemptAt == nil || secondCheckpoint.LastSyncSuccessAt == nil ||
		secondCheckpoint.LastCleanSyncAt == nil {
		t.Fatalf("second sync checkpoint = %+v, want empty normal acknowledgement", secondCheckpoint)
	}

	hostStateResponse, err := server.Admin.GetHostSantaStateWithResponse(t.Context(), hostID)
	hostStateResponse = requireAPIResponse(t, "get host Santa state", http.StatusOK, hostStateResponse, err)
	if hostStateResponse.JSON200 == nil {
		t.Fatal("get host Santa state returned no JSON body")
	}
	hostState := *hostStateResponse.JSON200
	if hostState.Version != "2026.7" || hostState.ClientModeReported != "monitor" ||
		hostState.Configuration == nil || hostState.Configuration.Id != configuration.Id ||
		hostState.Configuration.Name != configuration.Name || hostState.Configuration.MatchedViaLabel == nil ||
		hostState.Configuration.MatchedViaLabel.Id != label.Id ||
		hostState.Configuration.MatchedViaLabel.Name != label.Name ||
		hostState.RuleSync.DesiredCount != 1 || hostState.RuleSync.AppliedCount != 1 ||
		hostState.RuleSync.PendingCount != 0 || hostState.RuleSync.LastCleanSyncAt == nil {
		t.Fatalf("public Santa host state = %+v, want matched applied configuration", hostState)
	}

	hostRulesResponse, err := server.Admin.ListHostSantaRulesWithResponse(
		t.Context(),
		hostID,
		&adminapi.ListHostSantaRulesParams{PerPage: new(int32(1000))},
	)
	hostRulesResponse = requireAPIResponse(t, "list host Santa rules", http.StatusOK, hostRulesResponse, err)
	if hostRulesResponse.JSON200 == nil {
		t.Fatal("list host Santa rules returned no JSON body")
	}
	hostRules := *hostRulesResponse.JSON200
	if hostRules.Count != 1 || len(hostRules.Items) != 1 {
		t.Fatalf("public Santa host rules = %+v, want one", hostRules)
	}
	hostRule := hostRules.Items[0]
	if hostRule.RuleId != createdRule.Id || hostRule.RuleType != "binary" ||
		hostRule.Identifier != ruleIdentifier || hostRule.Policy != "blocklist" ||
		hostRule.CustomMessage == nil || *hostRule.CustomMessage != ruleMessage ||
		hostRule.CustomUrl == nil || *hostRule.CustomUrl != ruleURL || !hostRule.Applied {
		t.Fatalf("public Santa host rule = %+v, want applied integration rule", hostRule)
	}

	eventsResponse, err := server.Admin.ListSantaEventsWithResponse(
		t.Context(),
		&adminapi.ListSantaEventsParams{HostId: new(hostID), PerPage: new(int32(10))},
	)
	eventsResponse = requireAPIResponse(t, "list Santa events", http.StatusOK, eventsResponse, err)
	if eventsResponse.JSON200 == nil {
		t.Fatal("list Santa events returned no JSON body")
	}
	events := *eventsResponse.JSON200
	if events.Count != 1 || len(events.Items) != 1 {
		t.Fatalf("public Santa events = %+v, want one", events)
	}
	event := events.Items[0]
	if event.HostId != hostID || event.FilePath != eventPath || event.ExecutingUser != "alice" ||
		event.Pid != 4242 || event.Ppid != 1 || event.ParentName != "launchd" ||
		event.Decision != "block_binary" || !event.StaticRule || !event.OccurredAt.Equal(eventTime) ||
		event.Executable.Sha256 != eventSHA256 || event.Executable.FileName != "Woodstar Santa Test" ||
		event.Executable.SigningId != "ABCDE12345:au.edu.woodleigh.woodstar.santa-test" ||
		event.Executable.TeamId != "ABCDE12345" || event.Executable.Cdhash != eventCDHash ||
		event.Executable.SigningStatus != "production" {
		t.Fatalf("public Santa event = %+v, want uploaded execution evidence", event)
	}

	var observedMachineID, observedSerial, observedVersion, observedMode, primaryUser string
	var primaryGroups []string
	var sipStatus int16
	var lastSeenAt *time.Time
	if err := database.QueryRow(t.Context(), `
SELECT machine_id, serial_number, santa_version, client_mode_reported::text,
       primary_user, primary_user_groups, sip_status, last_seen_at
FROM santa_hosts
WHERE host_id = $1`, hostID).Scan(
		&observedMachineID,
		&observedSerial,
		&observedVersion,
		&observedMode,
		&primaryUser,
		&primaryGroups,
		&sipStatus,
		&lastSeenAt,
	); err != nil {
		t.Fatalf("load Santa host observation: %v", err)
	}
	if observedMachineID != machineID || observedSerial != serial || observedVersion != "2026.7" ||
		observedMode != "monitor" || primaryUser != "alice@santa.integration.test" ||
		!slices.Equal(primaryGroups, []string{"students", "santa-integration"}) ||
		sipStatus != 1 || lastSeenAt == nil {
		t.Fatalf(
			"Santa host observation = %q/%q/%q/%q/%q/%v/%d/%v, want uploaded observation",
			observedMachineID,
			observedSerial,
			observedVersion,
			observedMode,
			primaryUser,
			primaryGroups,
			sipStatus,
			lastSeenAt,
		)
	}

	unknownClient := santaProtocolFixtureClient{
		t:         t,
		client:    server.Client,
		baseURL:   server.BaseURL,
		machineID: unknownMachineID,
		secret:    santaSecret,
	}
	unknownClient.postFixtureStatus(
		"preflight",
		"preflight_unknown.json",
		nil,
		&syncv1.PreflightRequest{},
		nil,
		http.StatusNotFound,
	)

	hostsAfterResponse, err := server.Admin.ListHostsWithResponse(
		t.Context(),
		&adminapi.ListHostsParams{PerPage: new(int32(1000))},
	)
	hostsAfterResponse = requireAPIResponse(
		t,
		"list hosts after unknown preflight",
		http.StatusOK,
		hostsAfterResponse,
		err,
	)
	if hostsAfterResponse.JSON200 == nil {
		t.Fatal("list hosts after unknown preflight returned no JSON body")
	}
	hostsAfter := *hostsAfterResponse.JSON200
	requireOnlySantaHost(t, hostsAfter, hostID, machineID, serial)
	if hostsAfter.Count != hostsBefore.Count {
		t.Fatalf("host count after unknown preflight = %d, want unchanged %d", hostsAfter.Count, hostsBefore.Count)
	}

	var unknownCanonicalCount, unknownObservationCount int
	if err := database.QueryRow(
		t.Context(),
		"SELECT count(*) FROM hosts WHERE hardware_uuid = $1",
		unknownMachineID,
	).Scan(&unknownCanonicalCount); err != nil {
		t.Fatalf("count unknown canonical hosts: %v", err)
	}
	if err := database.QueryRow(
		t.Context(),
		"SELECT count(*) FROM santa_hosts WHERE machine_id = $1",
		unknownMachineID,
	).Scan(&unknownObservationCount); err != nil {
		t.Fatalf("count unknown Santa observations: %v", err)
	}
	if unknownCanonicalCount != 0 || unknownObservationCount != 0 {
		t.Fatalf(
			"unknown machine canonical/observation counts = %d/%d, want 0/0",
			unknownCanonicalCount,
			unknownObservationCount,
		)
	}
}

func (client santaProtocolFixtureClient) postFixture(
	stage string,
	name string,
	values map[string]any,
	request proto.Message,
	response proto.Message,
) {
	client.t.Helper()
	client.postFixtureStatus(stage, name, values, request, response, http.StatusOK)
}

func (client santaProtocolFixtureClient) postFixtureStatus(
	stage string,
	name string,
	values map[string]any,
	request proto.Message,
	response proto.Message,
	wantStatus int,
) {
	client.t.Helper()

	payload := loadProtocolFixture(client.t, santaProtocolFixtures, "santa", name, values)
	if err := protojson.Unmarshal(payload, request); err != nil {
		client.t.Fatalf("unmarshal Santa protocol fixture %s: %v", name, err)
	}
	client.postProtoStatus(stage, request, response, wantStatus)
}

func (client santaProtocolFixtureClient) postProtoStatus(
	stage string,
	requestMessage proto.Message,
	responseMessage proto.Message,
	wantStatus int,
) {
	client.t.Helper()

	machineRequest, ok := requestMessage.(interface{ GetMachineId() string })
	if !ok || machineRequest.GetMachineId() != client.machineID {
		client.t.Fatalf("%s request machine ID does not match client", stage)
	}
	payload, err := proto.Marshal(requestMessage)
	if err != nil {
		client.t.Fatalf("marshal Santa %s request: %v", stage, err)
	}
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write(payload); err != nil {
		_ = zw.Close()
		client.t.Fatalf("compress Santa %s request: %v", stage, err)
	}
	if err := zw.Close(); err != nil {
		client.t.Fatalf("finish Santa %s request compression: %v", stage, err)
	}

	requestURL := client.baseURL + "/santa/sync/" + stage + "/" + client.machineID
	httpRequest, err := http.NewRequestWithContext(
		client.t.Context(),
		http.MethodPost,
		requestURL,
		bytes.NewReader(compressed.Bytes()),
	)
	if err != nil {
		client.t.Fatalf("create Santa %s request: %v", stage, err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+client.secret)
	httpRequest.Header.Set("Content-Type", santaProtobufContentType)
	httpRequest.Header.Set("Content-Encoding", "gzip")
	// Supplying Accept-Encoding ourselves keeps net/http from transparently
	// decoding and removing the response's Content-Encoding header.
	httpRequest.Header.Set("Accept-Encoding", "gzip")

	httpResponse, err := client.client.Do(httpRequest)
	if err != nil {
		client.t.Fatalf("send Santa %s request: %v", stage, err)
	}
	responseBody := readAndClose(client.t, httpResponse)
	if httpResponse.StatusCode != wantStatus {
		client.t.Fatalf("Santa %s status = %d, want %d", stage, httpResponse.StatusCode, wantStatus)
	}
	if wantStatus != http.StatusOK {
		if len(responseBody) != 0 {
			client.t.Fatalf("Santa %s error body length = %d, want empty", stage, len(responseBody))
		}
		if contentType := httpResponse.Header.Get("Content-Type"); contentType != "" {
			client.t.Fatalf("Santa %s error content type = %q, want absent", stage, contentType)
		}
		if contentEncoding := httpResponse.Header.Get("Content-Encoding"); contentEncoding != "" {
			client.t.Fatalf("Santa %s error content encoding = %q, want absent", stage, contentEncoding)
		}
		return
	}

	mediaType, _, err := mime.ParseMediaType(httpResponse.Header.Get("Content-Type"))
	if err != nil || mediaType != santaProtobufContentType {
		client.t.Fatalf(
			"Santa %s response content type = %q, want %q",
			stage,
			httpResponse.Header.Get("Content-Type"),
			santaProtobufContentType,
		)
	}
	if !strings.EqualFold(httpResponse.Header.Get("Content-Encoding"), "gzip") {
		client.t.Fatalf(
			"Santa %s response content encoding = %q, want gzip",
			stage,
			httpResponse.Header.Get("Content-Encoding"),
		)
	}
	zr, err := gzip.NewReader(bytes.NewReader(responseBody))
	if err != nil {
		client.t.Fatalf("open Santa %s response compression: %v", stage, err)
	}
	uncompressed, err := io.ReadAll(zr)
	if err != nil {
		_ = zr.Close()
		client.t.Fatalf("read Santa %s response: %v", stage, err)
	}
	if err := zr.Close(); err != nil {
		client.t.Fatalf("close Santa %s response compression: %v", stage, err)
	}
	if responseMessage == nil {
		client.t.Fatalf("Santa %s success requires a protobuf response target", stage)
	}
	if err := proto.Unmarshal(uncompressed, responseMessage); err != nil {
		client.t.Fatalf("unmarshal Santa %s response: %v", stage, err)
	}
}

func (client santaProtocolFixtureClient) downloadRules() []*syncv1.Rule {
	client.t.Helper()

	var rules []*syncv1.Rule
	cursor := ""
	for {
		var response syncv1.RuleDownloadResponse
		client.postFixture(
			"ruledownload",
			"rule_download.json",
			map[string]any{"$MACHINE_ID": client.machineID, "$CURSOR": cursor},
			&syncv1.RuleDownloadRequest{},
			&response,
		)
		rules = append(rules, response.GetRules()...)
		cursor = response.GetCursor()
		if cursor == "" {
			return rules
		}
	}
}

func requireSantaPreflightConfiguration(t *testing.T, response *syncv1.PreflightResponse) {
	t.Helper()

	if response.SyncType == nil || response.GetSyncType() != syncv1.SyncType_CLEAN {
		t.Fatalf("first sync type = %v, want present CLEAN", response.SyncType)
	}
	if response.GetClientMode() != syncv1.ClientMode_LOCKDOWN ||
		response.GetFullSyncIntervalSeconds() != 600 || response.GetBatchSize() != 50 {
		t.Fatalf(
			"configured client mode/interval/batch = %v/%d/%d, want LOCKDOWN/600/50",
			response.GetClientMode(),
			response.GetFullSyncIntervalSeconds(),
			response.GetBatchSize(),
		)
	}
	if response.EnableBundles == nil || !response.GetEnableBundles() ||
		response.EnableTransitiveRules == nil || !response.GetEnableTransitiveRules() ||
		response.EnableAllEventUpload == nil || !response.GetEnableAllEventUpload() ||
		response.DisableUnknownEventUpload == nil || !response.GetDisableUnknownEventUpload() {
		t.Fatalf("configured optional event booleans = %+v, want all present and true", response)
	}
	if response.OverrideFileAccessAction == nil ||
		response.GetOverrideFileAccessAction() != syncv1.FileAccessAction_AUDIT_ONLY {
		t.Fatalf("override file access action = %v, want present AUDIT_ONLY", response.OverrideFileAccessAction)
	}
	if response.AllowedPathRegex == nil || response.GetAllowedPathRegex() != `^/Applications/` ||
		response.BlockedPathRegex == nil || response.GetBlockedPathRegex() != `^/tmp/` {
		t.Fatalf(
			"configured path regexes = %v/%v, want present integration values",
			response.AllowedPathRegex,
			response.BlockedPathRegex,
		)
	}
	if response.EventDetailUrl == nil ||
		response.GetEventDetailUrl() != "https://woodstar.example.test/santa" ||
		response.EventDetailText == nil || response.GetEventDetailText() != "More information" {
		t.Fatalf(
			"configured event detail = %v/%v, want present integration values",
			response.EventDetailUrl,
			response.EventDetailText,
		)
	}
}

func requireOnlySantaHost(
	t *testing.T,
	page adminapi.PageHost,
	hostID int64,
	machineID string,
	serial string,
) {
	t.Helper()

	if page.Count != 1 || len(page.Items) != 1 {
		t.Fatalf("public hosts = %+v, want only seeded Santa host", page)
	}
	host := page.Items[0]
	if host.Id != hostID || host.DisplayName != "Santa Integration Mac" ||
		host.Hardware.Uuid != machineID || host.Hardware.Serial != serial {
		t.Fatalf("public host = %+v, want seeded Santa host", host)
	}
}

func loadSantaSyncCheckpoint(t *testing.T, database *pgx.Conn, hostID int64) santaSyncCheckpoint {
	t.Helper()

	var checkpoint santaSyncCheckpoint
	err := database.QueryRow(t.Context(), `
SELECT
    rules_received,
    rules_processed,
    confirmed_rules_hash,
    pending_full_sync,
    pending_payload_rule_count,
    pending_preflight_at,
    last_rule_sync_attempt_at,
    last_rule_sync_success_at,
    last_clean_sync_at,
    (SELECT count(*)::integer
     FROM santa_sync_targets
     WHERE host_id = $1 AND phase = 'desired') AS desired_target_count,
    (SELECT count(*)::integer
     FROM santa_sync_targets
     WHERE host_id = $1 AND phase = 'applied') AS applied_target_count
FROM santa_sync_state
WHERE host_id = $1`, hostID).Scan(
		&checkpoint.RulesReceived,
		&checkpoint.RulesProcessed,
		&checkpoint.ConfirmedRulesHash,
		&checkpoint.PendingFullSync,
		&checkpoint.PendingPayloadCount,
		&checkpoint.PendingPreflightAt,
		&checkpoint.LastSyncAttemptAt,
		&checkpoint.LastSyncSuccessAt,
		&checkpoint.LastCleanSyncAt,
		&checkpoint.DesiredTargetCount,
		&checkpoint.AppliedTargetCount,
	)
	if err != nil {
		t.Fatalf("load Santa sync checkpoint: %v", err)
	}
	return checkpoint
}

func (checkpoint santaSyncCheckpoint) String() string {
	return fmt.Sprintf(
		"received=%d processed=%d hash=%q pending_full=%t pending_count=%d pending_at=%v attempt=%v success=%v clean=%v desired=%d applied=%d",
		checkpoint.RulesReceived,
		checkpoint.RulesProcessed,
		checkpoint.ConfirmedRulesHash,
		checkpoint.PendingFullSync,
		checkpoint.PendingPayloadCount,
		checkpoint.PendingPreflightAt,
		checkpoint.LastSyncAttemptAt,
		checkpoint.LastSyncSuccessAt,
		checkpoint.LastCleanSyncAt,
		checkpoint.DesiredTargetCount,
		checkpoint.AppliedTargetCount,
	)
}
