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
)

func TestSantaSyncRoutesDecodeAndEncodeGzippedProtobuf(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		request   proto.Message
		response  proto.Message
		wantStage string
	}{
		{
			name:      "preflight",
			path:      "/api/santa/sync/preflight/machine-1",
			request:   &syncv1.PreflightRequest{},
			response:  &syncv1.PreflightResponse{},
			wantStage: "preflight",
		},
		{
			name:      "event upload",
			path:      "/api/santa/sync/eventupload/machine-1",
			request:   &syncv1.EventUploadRequest{},
			response:  &syncv1.EventUploadResponse{},
			wantStage: "eventupload",
		},
		{
			name:      "rule download",
			path:      "/api/santa/sync/ruledownload/machine-1",
			request:   &syncv1.RuleDownloadRequest{},
			response:  &syncv1.RuleDownloadResponse{},
			wantStage: "ruledownload",
		},
		{
			name:      "postflight",
			path:      "/api/santa/sync/postflight/machine-1",
			request:   &syncv1.PostflightRequest{},
			response:  &syncv1.PostflightResponse{},
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
			if got := rec.Header().Get("Content-Type"); got != protobufContentType {
				t.Fatalf("content type = %q, want %q", got, protobufContentType)
			}
			if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
				t.Fatalf("content encoding = %q, want gzip", got)
			}
			mustUnzipProto(t, rec.Body.Bytes(), tt.response)
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
		tokenVerifier   TokenVerifier
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
			contentType:     "application/json",
			contentEncoding: "gzip",
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
			req := httptest.NewRequest(http.MethodPost, "/api/santa/sync/preflight/machine-1", bytes.NewReader(tt.body))
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

func testRouter(verifier TokenVerifier, service Service) chi.Router {
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

func mustUnzipProto(t *testing.T, payload []byte, msg proto.Message) {
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
	if err := proto.Unmarshal(decoded, msg); err != nil {
		t.Fatalf("unmarshal proto: %v", err)
	}
}

type staticVerifier struct {
	ok  bool
	err error
}

func (v *staticVerifier) VerifyBearerToken(context.Context, string) (bool, error) {
	return v.ok, v.err
}

type recordingService struct {
	stage     string
	machineID string
}

func (s *recordingService) HandlePreflight(
	_ context.Context,
	machineID string,
	_ *syncv1.PreflightRequest,
) (*syncv1.PreflightResponse, error) {
	s.stage = "preflight"
	s.machineID = machineID
	return &syncv1.PreflightResponse{}, nil
}

func (s *recordingService) HandleEventUpload(
	_ context.Context,
	machineID string,
	_ *syncv1.EventUploadRequest,
) (*syncv1.EventUploadResponse, error) {
	s.stage = "eventupload"
	s.machineID = machineID
	return &syncv1.EventUploadResponse{}, nil
}

func (s *recordingService) HandleRuleDownload(
	_ context.Context,
	machineID string,
	_ *syncv1.RuleDownloadRequest,
) (*syncv1.RuleDownloadResponse, error) {
	s.stage = "ruledownload"
	s.machineID = machineID
	return &syncv1.RuleDownloadResponse{}, nil
}

func (s *recordingService) HandlePostflight(
	_ context.Context,
	machineID string,
	_ *syncv1.PostflightRequest,
) (*syncv1.PostflightResponse, error) {
	s.stage = "postflight"
	s.machineID = machineID
	return &syncv1.PostflightResponse{}, nil
}
