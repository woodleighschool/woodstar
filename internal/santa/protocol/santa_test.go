package protocol

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/proto"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

func TestSantaSyncRoutesDecodeAndEncodeRuleDownloadCursor(t *testing.T) {
	service := &recordingService{ruleDownloadResponse: syncstate.RuleDownloadResponse{Cursor: "next"}}
	router := testRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaSyncRequest(t, "/santa/sync/ruledownload/machine-1",
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

func TestSantaSyncRoutesCoverAllStages(t *testing.T) {
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
			router := testRouter(&staticVerifier{ok: true}, service)
			rec := httptest.NewRecorder()
			req := santaSyncRequest(t, tt.path, tt.request)

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

func TestSantaSyncRoutesRejectAgentErrorsWithEmptyBodies(t *testing.T) {
	validBody := mustGzipProto(t, &syncv1.PreflightRequest{})
	malformedProto := mustGzip(t, []byte("not a protobuf"))

	tests := []struct {
		name            string
		tokenVerifier   SyncTokenVerifier
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
			router := testRouter(tt.tokenVerifier, &recordingService{})
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

func TestSantaSyncRoutesMapInvalidCursorToBadRequest(t *testing.T) {
	service := &recordingService{err: dbutil.ErrInvalidInput}
	router := testRouter(&staticVerifier{ok: true}, service)
	rec := httptest.NewRecorder()
	req := santaSyncRequest(t, "/santa/sync/ruledownload/machine-1",
		&syncv1.RuleDownloadRequest{Cursor: "bad"})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func testRouter(verifier SyncTokenVerifier, service SyncService) chi.Router {
	r := chi.NewRouter()
	RegisterSantaRoutes(r, verifier, service, slog.New(slog.DiscardHandler))
	return r
}

func santaSyncRequest(t *testing.T, path string, msg proto.Message) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(mustGzipProto(t, msg)))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", protobufContentType)
	req.Header.Set("Content-Encoding", "gzip")
	return req
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

func (v *staticVerifier) VerifySyncToken(context.Context, string) (bool, error) {
	return v.ok, v.err
}

type recordingService struct {
	stage                string
	machineID            string
	ruleDownloadCursor   string
	ruleDownloadResponse syncstate.RuleDownloadResponse
	err                  error
}

func (s *recordingService) Preflight(
	_ context.Context,
	machineID string,
	_ syncstate.PreflightRequest,
) (syncstate.PreflightResponse, error) {
	s.stage = "preflight"
	s.machineID = machineID
	return syncstate.PreflightResponse{}, s.err
}

func (s *recordingService) EventUpload(
	_ context.Context,
	machineID string,
	_ syncstate.EventUploadRequest,
) (syncstate.EventUploadResponse, error) {
	s.stage = "eventupload"
	s.machineID = machineID
	return syncstate.EventUploadResponse{}, s.err
}

func (s *recordingService) RuleDownload(
	_ context.Context,
	machineID string,
	req syncstate.RuleDownloadRequest,
) (syncstate.RuleDownloadResponse, error) {
	s.stage = "ruledownload"
	s.machineID = machineID
	s.ruleDownloadCursor = req.Cursor
	return s.ruleDownloadResponse, s.err
}

func (s *recordingService) Postflight(
	_ context.Context,
	machineID string,
	_ syncstate.PostflightRequest,
) (syncstate.PostflightResponse, error) {
	s.stage = "postflight"
	s.machineID = machineID
	return syncstate.PostflightResponse{}, s.err
}
