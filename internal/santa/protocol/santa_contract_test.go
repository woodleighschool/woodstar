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
	"strings"
	"testing"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/proto"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

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

func TestSantaHTTPEventUploadMapsStaticAndAuditEvents(t *testing.T) {
	service := &recordingService{}
	router := newSantaContractRouter(&staticVerifier{ok: true}, service)
	createdAt := time.Date(2026, 7, 14, 9, 30, 0, 0, time.UTC)
	rec := httptest.NewRecorder()
	req := santaContractRequest(t, "/santa/sync/eventupload/machine-1", &syncv1.EventUploadRequest{
		Events: []*syncv1.Event{{
			FileSha256:    strings.Repeat("1", 64),
			Decision:      syncv1.Decision_BLOCK_BINARY,
			ExecutionTime: float64(createdAt.Unix()),
			StaticRule:    true,
		}},
		AuditEvents: []*syncv1.AuditEvent{{
			Event: &syncv1.AuditEvent_StandaloneModeRuleCreation{
				StandaloneModeRuleCreation: &syncv1.StandaloneModeRuleCreation{
					Decision:   syncv1.Decision_ALLOW_BINARY,
					Identifier: strings.Repeat("2", 64),
					Timestamp:  uint32(createdAt.Unix()),
				},
			},
		}},
	})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	got := service.eventUploadRequest
	if len(got.Events) != 1 || !got.Events[0].StaticRule {
		t.Fatalf("execution events = %+v, want static rule", got.Events)
	}
	if len(got.StandaloneRuleCreationEvents) != 1 {
		t.Fatalf("audit events = %+v, want one", got.StandaloneRuleCreationEvents)
	}
	audit := got.StandaloneRuleCreationEvents[0]
	if audit.Identifier != strings.Repeat("2", 64) ||
		audit.Decision != santaevents.ExecutionDecisionAllowBinary ||
		!audit.OccurredAt.Equal(createdAt) {
		t.Fatalf("audit event = %+v", audit)
	}
}

func TestSantaHTTPRejectsUnsupportedAuditEvents(t *testing.T) {
	service := &recordingService{}
	router := newSantaContractRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaContractRequest(t, "/santa/sync/eventupload/machine-1", &syncv1.EventUploadRequest{
		AuditEvents: []*syncv1.AuditEvent{{}},
	})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if service.stage != "" {
		t.Fatalf("service stage = %q, want no call", service.stage)
	}
}

func TestSantaHTTPPostflightRequiresCurrentSyncType(t *testing.T) {
	for _, syncType := range []syncv1.SyncType{
		syncv1.SyncType_SYNC_TYPE_UNSPECIFIED,
		syncv1.SyncType_CLEAN_STANDALONE,
	} {
		t.Run(syncType.String(), func(t *testing.T) {
			service := &recordingService{}
			router := newSantaContractRouter(&staticVerifier{ok: true}, service)
			rec := httptest.NewRecorder()
			req := santaContractRequest(t, "/santa/sync/postflight/machine-1", &syncv1.PostflightRequest{
				SyncType:  syncType,
				RulesHash: strings.Repeat("0", 32),
			})

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if service.stage != "" {
				t.Fatalf("service stage = %q, want no call", service.stage)
			}
		})
	}
}

func TestSantaHTTPPostflightMapsSyncContract(t *testing.T) {
	for _, tc := range []struct {
		proto syncv1.SyncType
		want  syncstate.SyncType
	}{
		{proto: syncv1.SyncType_CLEAN, want: syncstate.SyncTypeClean},
		{proto: syncv1.SyncType_CLEAN_ALL, want: syncstate.SyncTypeCleanAll},
	} {
		t.Run(tc.proto.String(), func(t *testing.T) {
			service := &recordingService{}
			router := newSantaContractRouter(&staticVerifier{ok: true}, service)
			rec := httptest.NewRecorder()
			req := santaContractRequest(t, "/santa/sync/postflight/machine-1", &syncv1.PostflightRequest{
				RulesReceived:  3,
				RulesProcessed: 3,
				SyncType:       tc.proto,
				RulesHash:      strings.Repeat("a", 32),
			})

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			want := santa.PostflightRequest{
				RulesReceived:  3,
				RulesProcessed: 3,
				SyncType:       tc.want,
				RulesHash:      strings.Repeat("a", 32),
			}
			if service.postflightRequest != want {
				t.Fatalf("postflight request = %+v, want %+v", service.postflightRequest, want)
			}
		})
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

func TestSantaHTTPRejectsMachineIDMismatch(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		request proto.Message
	}{
		{
			name:    "preflight",
			path:    "/santa/sync/preflight/machine-1",
			request: &syncv1.PreflightRequest{MachineId: "machine-2"},
		},
		{
			name:    "event upload",
			path:    "/santa/sync/eventupload/machine-1",
			request: &syncv1.EventUploadRequest{MachineId: "machine-2"},
		},
		{
			name:    "rule download",
			path:    "/santa/sync/ruledownload/machine-1",
			request: &syncv1.RuleDownloadRequest{MachineId: "machine-2"},
		},
		{
			name:    "postflight",
			path:    "/santa/sync/postflight/machine-1",
			request: &syncv1.PostflightRequest{MachineId: "machine-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &recordingService{}
			router := newSantaContractRouter(&staticVerifier{ok: true}, service)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(mustGzipProto(t, tt.request)))
			req.Header.Set("Authorization", "Bearer ok")
			req.Header.Set("Content-Type", protobufContentType)
			req.Header.Set("Content-Encoding", "gzip")

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if service.stage != "" {
				t.Fatalf("service stage = %q, want no service call", service.stage)
			}
		})
	}
}

func TestSantaHTTPRejectsAgentErrorsWithEmptyBodies(t *testing.T) {
	validBody := mustGzipProto(t, &syncv1.PreflightRequest{MachineId: "machine-1"})
	malformedProto := mustGzip(t, []byte("not a protobuf"))

	tests := []struct {
		name            string
		tokenVerifier   agentauth.SecretVerifier
		body            []byte
		contentType     string
		contentEncoding string
		authorization   string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newSantaContractRouter(tt.tokenVerifier, &recordingService{})
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
	NewServer(verifier, service, slog.New(slog.DiscardHandler)).RegisterRoutes(r)
	return r
}

func santaContractRequest(t *testing.T, path string, msg proto.Message) *http.Request {
	t.Helper()

	setter, ok := msg.(interface{ SetMachineId(machineID string) })
	if !ok {
		t.Fatalf("request %T cannot set machine_id", msg)
	}
	setter.SetMachineId(machineIDFromSyncPath(path))

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(mustGzipProto(t, msg)))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", protobufContentType)
	req.Header.Set("Content-Encoding", "gzip")
	return req
}

func machineIDFromSyncPath(path string) string {
	return path[strings.LastIndex(path, "/")+1:]
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
	postflightRequest    santa.PostflightRequest
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
	return santa.PreflightResponse{SyncType: syncstate.SyncTypeNormal}, s.err
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
	req santa.PostflightRequest,
) (santa.PostflightResponse, error) {
	s.stage = "postflight"
	s.machineID = machineID
	s.postflightRequest = req
	return santa.PostflightResponse{}, s.err
}
