package protocol

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/proto"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestSantaHTTPPreflightRuleDownloadPostflightAndEventUpload(t *testing.T) {
	db, ctx := dbtest.Open(t)
	stores := newSantaContractStores(db)
	router := newSantaIntegratedContractRouter(stores)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	machineID := "santa-contract-" + suffix
	host, err := stores.hosts.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware: hosts.HostHardware{
			UUID:   machineID,
			Serial: "SANTACONTRACT",
		},
		OrbitNodeKey: "santa-contract-orbit-" + suffix,
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	label, err := stores.labels.Create(ctx, labels.LabelMutation{
		Name:                "Santa Contract " + suffix,
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	if err := stores.labels.SetMembership(ctx, label.ID, host.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}

	if _, err := stores.configurations.Create(ctx, configurations.ConfigurationMutation{
		Name:                    "Contract configuration " + suffix,
		ClientMode:              configurations.ClientModeLockdown,
		EnableBundles:           true,
		FullSyncIntervalSeconds: 600,
		BatchSize:               50,
		Targets: configurations.ConfigurationTargets{
			Include: []targeting.LabelRef{{LabelID: label.ID}},
		},
	}); err != nil {
		t.Fatalf("create configuration: %v", err)
	}

	ruleIdentifier := strings.Repeat("a", 64)
	if _, err := stores.rules.CreateRule(ctx, santarules.RuleMutation{
		RuleType:      santarules.RuleTypeBinary,
		Identifier:    ruleIdentifier,
		Name:          "Contract rule " + suffix,
		CustomMessage: "Blocked by contract",
		Targets: santarules.RuleTargets{
			Include: []santarules.RuleInclude{{
				Policy:  santarules.PolicyBlocklist,
				LabelID: label.ID,
			}},
		},
	}); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	secret, err := stores.agentSecrets.Create(
		ctx,
		agentauth.AgentSecretCreate{Agent: agentauth.AgentSanta, Value: "santa-contract-secret-value-long-32"},
	)
	if err != nil {
		t.Fatalf("create santa agent secret: %v", err)
	}

	var preflight syncv1.PreflightResponse
	doSantaContractProto(t, router, secret.Value, "/santa/sync/preflight/"+machineID, &syncv1.PreflightRequest{
		SerialNumber:     "SANTACONTRACT",
		SantaVersion:     "2026.2",
		ClientMode:       syncv1.ClientMode_MONITOR,
		RequestCleanSync: true,
		RulesHash:        "client-hash-before",
	}, &preflight)
	if preflight.GetSyncType() != syncv1.SyncType_CLEAN {
		t.Fatalf("sync type = %v, want CLEAN", preflight.GetSyncType())
	}
	if preflight.GetClientMode() != syncv1.ClientMode_LOCKDOWN {
		t.Fatalf("client mode = %v, want LOCKDOWN", preflight.GetClientMode())
	}
	if preflight.EnableBundles == nil || !preflight.GetEnableBundles() {
		t.Fatalf("enable bundles = %v, want true", preflight.EnableBundles)
	}

	var download syncv1.RuleDownloadResponse
	doSantaContractProto(t, router, secret.Value, "/santa/sync/ruledownload/"+machineID,
		&syncv1.RuleDownloadRequest{}, &download)
	if len(download.GetRules()) != 1 {
		t.Fatalf("downloaded rules = %+v, want one", download.GetRules())
	}
	rule := download.GetRules()[0]
	if rule.GetIdentifier() != ruleIdentifier ||
		rule.GetRuleType() != syncv1.RuleType_BINARY ||
		rule.GetPolicy() != syncv1.Policy_BLOCKLIST ||
		rule.GetCustomMsg() != "Blocked by contract" {
		t.Fatalf("downloaded rule = %+v", rule)
	}

	doSantaContractProto(t, router, secret.Value, "/santa/sync/postflight/"+machineID, &syncv1.PostflightRequest{
		RulesHash:      "client-hash-after",
		RulesReceived:  uint32(len(download.GetRules())),
		RulesProcessed: uint32(len(download.GetRules())),
	}, &syncv1.PostflightResponse{})

	doSantaContractProto(t, router, secret.Value, "/santa/sync/eventupload/"+machineID, &syncv1.EventUploadRequest{
		Events: []*syncv1.Event{
			{
				FileSha256:    "sha256-contract-" + suffix,
				FilePath:      "/Applications/Contract.app/Contents/MacOS/Contract",
				FileName:      "Contract",
				ExecutingUser: "alice",
				Decision:      syncv1.Decision_BLOCK_BINARY,
				ExecutionTime: float64(time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC).Unix()),
			},
			{
				FileSha256:    "sha256-mismatch-" + suffix,
				FilePath:      "/Applications/Mismatch.app/Contents/MacOS/Mismatch",
				FileName:      "Mismatch",
				ExecutingUser: "alice",
				Decision:      syncv1.Decision_BLOCK_BINARY_MISMATCH,
				ExecutionTime: float64(time.Date(2026, 5, 24, 10, 1, 0, 0, time.UTC).Unix()),
			},
			{
				FileSha256:    "sha256-platform-" + suffix,
				FilePath:      "/System/Library/CoreServices/SystemUIServer.app/Contents/MacOS/SystemUIServer",
				FileName:      "SystemUIServer",
				ExecutingUser: "alice",
				Decision:      syncv1.Decision_ALLOW_PLATFORM,
				ExecutionTime: float64(time.Date(2026, 5, 24, 10, 2, 0, 0, time.UTC).Unix()),
			},
		},
		FileAccessEvents: []*syncv1.FileAccessEvent{{
			RuleVersion: "policy-v1",
			RuleName:    "Protect Payroll",
			Target:      "/Users/alice/Payroll.csv",
			Decision:    syncv1.FileAccessDecision_FILE_ACCESS_DECISION_DENIED,
			AccessTime:  float64(time.Date(2026, 5, 24, 10, 5, 0, 0, time.UTC).Unix()),
			ProcessChain: []*syncv1.Process{{
				Pid:        123,
				FilePath:   "/Applications/Contract.app/Contents/MacOS/Contract",
				FileSha256: "process-sha-contract-" + suffix,
				SigningId:  "TEAMID:contract",
				TeamId:     "TEAMID",
				Cdhash:     "process-cdhash",
			}},
		}},
	}, &syncv1.EventUploadResponse{})

	events, _, err := stores.events.ListEvents(ctx, santaevents.ExecutionEventListParams{
		EventListParams: santaevents.EventListParams{HostID: host.ID},
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	wantDecisions := map[santaevents.ExecutionDecision]bool{
		santaevents.ExecutionDecisionBlockBinary:         true,
		santaevents.ExecutionDecisionBlockBinaryMismatch: true,
		santaevents.ExecutionDecisionAllowPlatform:       true,
	}
	if len(events) != len(wantDecisions) {
		t.Fatalf("stored events = %+v, want %d execution events", events, len(wantDecisions))
	}
	for _, event := range events {
		if !wantDecisions[event.Decision] {
			t.Fatalf("stored decision = %q, want one of %+v", event.Decision, wantDecisions)
		}
		delete(wantDecisions, event.Decision)
	}
	if len(wantDecisions) != 0 {
		t.Fatalf("missing decisions = %+v", wantDecisions)
	}

	fileAccessEvents, _, err := stores.events.ListFileAccessEvents(ctx, santaevents.FileAccessEventListParams{
		EventListParams: santaevents.EventListParams{HostID: host.ID},
	})
	if err != nil {
		t.Fatalf("list file access events: %v", err)
	}
	if len(fileAccessEvents) != 1 ||
		fileAccessEvents[0].Decision != santaevents.FileAccessDecisionDenied ||
		fileAccessEvents[0].Target != "/Users/alice/Payroll.csv" ||
		fileAccessEvents[0].PrimaryProcess.FileSHA256 != "process-sha-contract-"+suffix {
		t.Fatalf("stored file access events = %+v, want one denied payroll event", fileAccessEvents)
	}
}

func TestSantaHTTPRuleDownloadRoundTripsCursor(t *testing.T) {
	service := &recordingService{ruleDownloadResponse: santa.RuleDownloadResponse{Cursor: "next"}}
	router := newSantaContractRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaContractRequest(t, "/santa/sync/ruledownload/machine-1",
		&syncv1.RuleDownloadRequest{Cursor: "current"})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != protobufContentType {
		t.Fatalf("content type = %q, want %q", got, protobufContentType)
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("content encoding = %q, want gzip", got)
	}
	var resp syncv1.RuleDownloadResponse
	mustReadProtoResponse(t, rec.Body.Bytes(), &resp)
	if resp.GetCursor() != "next" {
		t.Fatalf("cursor = %q, want next", resp.GetCursor())
	}
	if service.stage != "ruledownload" || service.machineID != "machine-1" {
		t.Fatalf("stage/machine = %q/%q", service.stage, service.machineID)
	}
	if service.ruleDownloadCursor != "current" {
		t.Fatalf("request cursor = %q, want current", service.ruleDownloadCursor)
	}
}

func TestSantaHTTPAcceptsProtobufMediaTypeParameters(t *testing.T) {
	service := &recordingService{}
	router := newSantaContractRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaContractRequest(t, "/santa/sync/preflight/machine-1", &syncv1.PreflightRequest{})
	req.Header.Set("Content-Type", protobufContentType+"; charset=utf-8")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestSantaHTTPPreflightDecodesRuleCounts(t *testing.T) {
	service := &recordingService{}
	router := newSantaContractRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaContractRequest(t, "/santa/sync/preflight/machine-1", &syncv1.PreflightRequest{
		BinaryRuleCount:      1,
		CertificateRuleCount: 2,
		TeamidRuleCount:      3,
		SigningidRuleCount:   4,
		CdhashRuleCount:      5,
		CompilerRuleCount:    6,
		TransitiveRuleCount:  7,
	})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if service.preflightCounts != (syncstate.RuleCounts{
		Binary:      1,
		Certificate: 2,
		TeamID:      3,
		SigningID:   4,
		CDHash:      5,
		Compiler:    6,
		Transitive:  7,
	}) {
		t.Fatalf("rule counts = %+v", service.preflightCounts)
	}
}

func TestSantaHTTPEventUploadMapsBundleFieldsAndEncodesBundleRequests(t *testing.T) {
	bundleHash := strings.Repeat("b", 64)
	service := &recordingService{
		eventUploadResponse: santa.EventUploadResponse{BundleBinaryRequests: []string{bundleHash}},
	}
	router := newSantaContractRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaContractRequest(t, "/santa/sync/eventupload/machine-1", &syncv1.EventUploadRequest{
		Events: []*syncv1.Event{{
			FileSha256:                  strings.Repeat("1", 64),
			FilePath:                    "/Applications/Bundle.app/Contents/MacOS/Helper",
			FileName:                    "Helper",
			Decision:                    syncv1.Decision_BUNDLE_BINARY,
			FileBundleId:                "com.example.bundle",
			FileBundlePath:              "/Applications/Bundle.app",
			FileBundleExecutableRelPath: "Contents/MacOS/Helper",
			FileBundleName:              "Bundle",
			FileBundleVersion:           "1.2.3",
			FileBundleVersionString:     "1.2.3 (45)",
			FileBundleHash:              bundleHash,
			FileBundleHashMillis:        17,
			FileBundleBinaryCount:       2,
			Pid:                         501,
			Ppid:                        1,
			ParentName:                  "launchd",
			SigningId:                   "TEAMID:com.example.bundle",
			TeamId:                      "TEAMID",
			Cdhash:                      "cdhash",
			CsFlags:                     570425345,
			SigningStatus:               syncv1.SigningStatus_SIGNING_STATUS_PRODUCTION,
			SecureSigningTime:           100,
			SigningTime:                 200,
		}},
	})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if service.stage != "eventupload" || service.machineID != "machine-1" {
		t.Fatalf("stage/machine = %q/%q", service.stage, service.machineID)
	}
	if len(service.eventUploadRequest.Events) != 1 {
		t.Fatalf("event upload request = %+v, want one event", service.eventUploadRequest)
	}
	event := service.eventUploadRequest.Events[0]
	if !event.OccurredAt.IsZero() ||
		event.Decision != santaevents.ExecutionDecisionBundleBinary ||
		event.BundleHash != bundleHash ||
		event.BundleBinaryCount != 2 ||
		event.BundleExecutableRelPath != "Contents/MacOS/Helper" ||
		event.PID != 501 ||
		event.PPID != 1 ||
		event.ParentName != "launchd" ||
		event.CodesigningFlags != 570425345 ||
		event.SigningStatus != santaevents.SigningStatusProduction ||
		!event.SecureSigningTime.Equal(time.Unix(100, 0).UTC()) ||
		!event.SigningTime.Equal(time.Unix(200, 0).UTC()) {
		t.Fatalf("mapped bundle event = %+v", event)
	}
	var resp syncv1.EventUploadResponse
	mustReadProtoResponse(t, rec.Body.Bytes(), &resp)
	if !slices.Equal(resp.GetEventUploadBundleBinaries(), []string{bundleHash}) {
		t.Fatalf("bundle binary response = %v, want [%s]", resp.GetEventUploadBundleBinaries(), bundleHash)
	}
}

func TestSantaHTTPRuleDownloadEncodesRemovedPayload(t *testing.T) {
	service := &recordingService{
		ruleDownloadResponse: santa.RuleDownloadResponse{
			Rules: []syncstate.PayloadRule{{
				RuleType:   "binary",
				Identifier: "old-rule",
				Removed:    true,
			}},
		},
	}
	router := newSantaContractRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaContractRequest(t, "/santa/sync/ruledownload/machine-1", &syncv1.RuleDownloadRequest{})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp syncv1.RuleDownloadResponse
	mustReadProtoResponse(t, rec.Body.Bytes(), &resp)
	if len(resp.GetRules()) != 1 {
		t.Fatalf("rules = %+v, want one", resp.GetRules())
	}
	if resp.GetRules()[0].GetPolicy() != syncv1.Policy_REMOVE {
		t.Fatalf("policy = %v, want REMOVE", resp.GetRules()[0].GetPolicy())
	}
}

func TestSantaHTTPRuleDownloadEncodesNotificationAppName(t *testing.T) {
	service := &recordingService{
		ruleDownloadResponse: santa.RuleDownloadResponse{
			Rules: []syncstate.PayloadRule{{
				RuleType:   "binary",
				Identifier: strings.Repeat("1", 64),
				Policy:     string(santarules.PolicyBlocklist),
				AppName:    "Bundle App",
			}},
		},
	}
	router := newSantaContractRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaContractRequest(t, "/santa/sync/ruledownload/machine-1", &syncv1.RuleDownloadRequest{})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp syncv1.RuleDownloadResponse
	mustReadProtoResponse(t, rec.Body.Bytes(), &resp)
	if len(resp.GetRules()) != 1 {
		t.Fatalf("rules = %+v, want one", resp.GetRules())
	}
	rule := resp.GetRules()[0]
	if rule.GetNotificationAppName() != "Bundle App" {
		t.Fatalf("notification_app_name = %q, want %q", rule.GetNotificationAppName(), "Bundle App")
	}
}

func TestSantaHTTPRuleDownloadEncodesSilentBlocklistPolicies(t *testing.T) {
	service := &recordingService{
		ruleDownloadResponse: santa.RuleDownloadResponse{
			Rules: []syncstate.PayloadRule{
				{
					RuleType:   "binary",
					Identifier: strings.Repeat("1", 64),
					Policy:     string(santarules.PolicySilentGUIBlocklist),
				},
				{
					RuleType:   "binary",
					Identifier: strings.Repeat("2", 64),
					Policy:     string(santarules.PolicySilentTTYBlocklist),
				},
			},
		},
	}
	router := newSantaContractRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaContractRequest(t, "/santa/sync/ruledownload/machine-1", &syncv1.RuleDownloadRequest{})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp syncv1.RuleDownloadResponse
	mustReadProtoResponse(t, rec.Body.Bytes(), &resp)
	if len(resp.GetRules()) != 2 {
		t.Fatalf("rules = %+v, want two", resp.GetRules())
	}
	if resp.GetRules()[0].GetPolicy() != syncv1.Policy_SILENT_GUI_BLOCKLIST {
		t.Fatalf("first policy = %v, want SILENT_GUI_BLOCKLIST", resp.GetRules()[0].GetPolicy())
	}
	if resp.GetRules()[1].GetPolicy() != syncv1.Policy_SILENT_TTY_BLOCKLIST {
		t.Fatalf("second policy = %v, want SILENT_TTY_BLOCKLIST", resp.GetRules()[1].GetPolicy())
	}
}

func TestSantaHTTPCoversAllSyncStages(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		request   proto.Message
		wantStage string
	}{
		{
			name:      "preflight",
			path:      "/santa/sync/preflight/machine-1",
			request:   &syncv1.PreflightRequest{},
			wantStage: "preflight",
		},
		{
			name:      "event upload",
			path:      "/santa/sync/eventupload/machine-1",
			request:   &syncv1.EventUploadRequest{},
			wantStage: "eventupload",
		},
		{
			name:      "rule download",
			path:      "/santa/sync/ruledownload/machine-1",
			request:   &syncv1.RuleDownloadRequest{},
			wantStage: "ruledownload",
		},
		{
			name:      "postflight",
			path:      "/santa/sync/postflight/machine-1",
			request:   &syncv1.PostflightRequest{},
			wantStage: "postflight",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &recordingService{}
			router := newSantaContractRouter(&staticVerifier{ok: true}, service)
			rec := httptest.NewRecorder()
			req := santaContractRequest(t, tt.path, tt.request)

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
			}
			if service.stage != tt.wantStage {
				t.Fatalf("stage = %q, want %q", service.stage, tt.wantStage)
			}
			if service.machineID != "machine-1" {
				t.Fatalf("machine id = %q, want machine-1", service.machineID)
			}
		})
	}
}

func TestSantaHTTPRejectsAgentErrorsWithEmptyBodies(t *testing.T) {
	validBody := mustGzipProto(t, &syncv1.PreflightRequest{})
	malformedProto := mustGzip(t, []byte("not a protobuf"))

	tests := []struct {
		name            string
		tokenVerifier   agentauth.SecretVerifier
		body            []byte
		contentType     string
		contentEncoding string
		authorization   string
		serviceErr      error
		wantStatus      int
	}{
		{
			name:            "missing bearer",
			tokenVerifier:   &staticVerifier{ok: true},
			body:            validBody,
			contentType:     protobufContentType,
			contentEncoding: "gzip",
			wantStatus:      http.StatusUnauthorized,
		},
		{
			name:            "wrong bearer scheme",
			tokenVerifier:   &staticVerifier{ok: true},
			body:            validBody,
			contentType:     protobufContentType,
			contentEncoding: "gzip",
			authorization:   "Token ok",
			wantStatus:      http.StatusUnauthorized,
		},
		{
			name:            "unknown token",
			tokenVerifier:   &staticVerifier{ok: false},
			body:            validBody,
			contentType:     protobufContentType,
			contentEncoding: "gzip",
			authorization:   "Bearer wrong",
			wantStatus:      http.StatusUnauthorized,
		},
		{
			name:            "wrong content type",
			tokenVerifier:   &staticVerifier{ok: true},
			body:            validBody,
			contentType:     "text/plain",
			contentEncoding: "gzip",
			authorization:   "Bearer ok",
			wantStatus:      http.StatusUnsupportedMediaType,
		},
		{
			name:            "unsupported encoding",
			tokenVerifier:   &staticVerifier{ok: true},
			body:            validBody,
			contentType:     protobufContentType,
			contentEncoding: "deflate",
			authorization:   "Bearer ok",
			wantStatus:      http.StatusUnsupportedMediaType,
		},
		{
			name:          "missing gzip encoding",
			tokenVerifier: &staticVerifier{ok: true},
			body:          validBody,
			contentType:   protobufContentType,
			authorization: "Bearer ok",
			wantStatus:    http.StatusUnsupportedMediaType,
		},
		{
			name:            "malformed gzip",
			tokenVerifier:   &staticVerifier{ok: true},
			body:            []byte("not gzip"),
			contentType:     protobufContentType,
			contentEncoding: "gzip",
			authorization:   "Bearer ok",
			wantStatus:      http.StatusBadRequest,
		},
		{
			name:            "malformed protobuf",
			tokenVerifier:   &staticVerifier{ok: true},
			body:            malformedProto,
			contentType:     protobufContentType,
			contentEncoding: "gzip",
			authorization:   "Bearer ok",
			wantStatus:      http.StatusBadRequest,
		},
		{
			name:            "unknown santa host",
			tokenVerifier:   &staticVerifier{ok: true},
			body:            validBody,
			contentType:     protobufContentType,
			contentEncoding: "gzip",
			authorization:   "Bearer ok",
			serviceErr:      dbutil.ErrNotFound,
			wantStatus:      http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newSantaContractRouter(tt.tokenVerifier, &recordingService{err: tt.serviceErr})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/santa/sync/preflight/machine-1", bytes.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if rec.Body.Len() != 0 {
				t.Fatalf("body = %q, want empty", rec.Body.String())
			}
		})
	}
}

func TestSantaHTTPAuthorizesOnlyActiveSantaAgentSecrets(t *testing.T) {
	db, ctx := dbtest.Open(t)
	secrets := agentauth.NewStore(db)
	service := &recordingService{}
	router := newSantaContractRouter(secrets, service)

	santaSecret, err := secrets.Create(
		ctx,
		agentauth.AgentSecretCreate{Agent: agentauth.AgentSanta, Value: "santa-active-secret-value-long-32"},
	)
	if err != nil {
		t.Fatalf("create santa agent secret: %v", err)
	}
	orbitSecret, err := secrets.Create(
		ctx,
		agentauth.AgentSecretCreate{Agent: agentauth.AgentOrbit, Value: "orbit-wrong-agent-secret-value-32"},
	)
	if err != nil {
		t.Fatalf("create orbit agent secret: %v", err)
	}
	deletedSecret, err := secrets.Create(
		ctx,
		agentauth.AgentSecretCreate{Agent: agentauth.AgentSanta, Value: "santa-deleted-secret-value-long-32"},
	)
	if err != nil {
		t.Fatalf("create deleted santa agent secret: %v", err)
	}
	if err := secrets.Delete(ctx, deletedSecret.ID); err != nil {
		t.Fatalf("delete santa agent secret: %v", err)
	}

	tests := []struct {
		name       string
		secret     string
		wantStatus int
	}{
		{name: "valid santa", secret: santaSecret.Value, wantStatus: http.StatusOK},
		{name: "orbit secret rejected", secret: orbitSecret.Value, wantStatus: http.StatusUnauthorized},
		{name: "deleted santa rejected", secret: deletedSecret.Value, wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := santaContractRequest(t, "/santa/sync/preflight/machine-1", &syncv1.PreflightRequest{})
			req.Header.Set("Authorization", "Bearer "+tt.secret)

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestSantaHTTPMapsInvalidCursorToBadRequest(t *testing.T) {
	service := &recordingService{err: dbutil.ErrInvalidInput}
	router := newSantaContractRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaContractRequest(t, "/santa/sync/ruledownload/machine-1",
		&syncv1.RuleDownloadRequest{Cursor: "bad"})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func newSantaContractRouter(verifier agentauth.SecretVerifier, service SyncService) chi.Router {
	r := chi.NewRouter()
	RegisterSantaRoutes(r, verifier, service, slog.New(slog.DiscardHandler))
	return r
}

type santaContractStores struct {
	hosts          *hosts.Store
	labels         *labels.Store
	hostState      *santa.Store
	agentSecrets   *agentauth.Store
	configurations *configurations.Store
	events         *santaevents.Store
	rules          *santarules.Store
	sync           *syncstate.Store
}

func newSantaContractStores(db *database.DB) santaContractStores {
	return santaContractStores{
		hosts:          hosts.NewStore(db),
		labels:         labels.NewStore(db),
		hostState:      santa.NewStore(db),
		agentSecrets:   agentauth.NewStore(db),
		configurations: configurations.NewStore(db),
		events:         santaevents.NewStore(db),
		rules:          santarules.NewStore(db),
		sync:           syncstate.NewStore(db),
	}
}

func newSantaIntegratedContractRouter(stores santaContractStores) chi.Router {
	service := santa.NewSyncService(santa.Dependencies{
		HostStore:      stores.hostState,
		Configurations: stores.configurations,
		Events:         stores.events,
		Rules:          stores.rules,
		Sync:           stores.sync,
	})
	return newSantaContractRouter(stores.agentSecrets, service)
}

func santaContractRequest(t *testing.T, path string, msg proto.Message) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(mustGzipProto(t, msg)))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", protobufContentType)
	req.Header.Set("Content-Encoding", "gzip")
	return req
}

func doSantaContractProto(
	t *testing.T,
	router http.Handler,
	token string,
	path string,
	request proto.Message,
	response proto.Message,
) {
	t.Helper()

	req := santaContractRequest(t, path, request)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if response != nil {
		mustReadProtoResponse(t, rec.Body.Bytes(), response)
	}
}

func mustGzipProto(t *testing.T, msg proto.Message) []byte {
	t.Helper()

	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal proto: %v", err)
	}
	return mustGzip(t, payload)
}

func mustGzip(t *testing.T, payload []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(payload); err != nil {
		t.Fatalf("write gzip: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}

func mustReadProtoResponse(t *testing.T, payload []byte, msg proto.Message) {
	t.Helper()

	payload = mustGunzip(t, payload)
	if err := proto.Unmarshal(payload, msg); err != nil {
		t.Fatalf("unmarshal proto: %v", err)
	}
}

func mustGunzip(t *testing.T, payload []byte) []byte {
	t.Helper()

	zr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("new gzip reader: %v", err)
	}
	defer zr.Close()
	decoded, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	return decoded
}

type staticVerifier struct {
	ok  bool
	err error
}

func (v *staticVerifier) Verify(context.Context, agentauth.Agent, string) (bool, error) {
	return v.ok, v.err
}

type recordingService struct {
	stage                string
	machineID            string
	preflightCounts      syncstate.RuleCounts
	eventUploadRequest   santa.EventUploadRequest
	eventUploadResponse  santa.EventUploadResponse
	ruleDownloadCursor   string
	ruleDownloadResponse santa.RuleDownloadResponse
	err                  error
}

func (s *recordingService) Preflight(
	_ context.Context,
	machineID string,
	req santa.PreflightRequest,
) (santa.PreflightResponse, error) {
	s.stage = "preflight"
	s.machineID = machineID
	s.preflightCounts = req.RuleCounts
	return santa.PreflightResponse{}, s.err
}

func (s *recordingService) EventUpload(
	_ context.Context,
	machineID string,
	req santa.EventUploadRequest,
) (santa.EventUploadResponse, error) {
	s.stage = "eventupload"
	s.machineID = machineID
	s.eventUploadRequest = req
	return s.eventUploadResponse, s.err
}

func (s *recordingService) RuleDownload(
	_ context.Context,
	machineID string,
	req santa.RuleDownloadRequest,
) (santa.RuleDownloadResponse, error) {
	s.stage = "ruledownload"
	s.machineID = machineID
	s.ruleDownloadCursor = req.Cursor
	return s.ruleDownloadResponse, s.err
}

func (s *recordingService) Postflight(
	_ context.Context,
	machineID string,
	_ santa.PostflightRequest,
) (santa.PostflightResponse, error) {
	s.stage = "postflight"
	s.machineID = machineID
	return santa.PostflightResponse{}, s.err
}
