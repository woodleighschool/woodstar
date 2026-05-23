// Package protocol exposes Santa sync endpoints.
package protocol

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/proto"

	"github.com/woodleighschool/woodstar/internal/santa"
)

const (
	protobufContentType = "application/x-protobuf"
	maxRequestBodyBytes = 16 << 20
)

var (
	errUnauthorized      = errors.New("unauthorized santa sync request")
	errUnsupportedMedia  = errors.New("unsupported santa sync media")
	errInvalidSyncBody   = errors.New("invalid santa sync request body")
	errRequestBodyTooBig = errors.New("santa sync request body too large")
)

// TokenVerifier verifies Santa bearer-token authorization headers.
type TokenVerifier interface {
	VerifyBearerToken(context.Context, string) (bool, error)
}

// Service handles decoded Santa sync requests.
type Service interface {
	HandlePreflight(context.Context, string, *syncv1.PreflightRequest) (*syncv1.PreflightResponse, error)
	HandleEventUpload(context.Context, string, *syncv1.EventUploadRequest) (*syncv1.EventUploadResponse, error)
	HandleRuleDownload(context.Context, string, *syncv1.RuleDownloadRequest) (*syncv1.RuleDownloadResponse, error)
	HandlePostflight(context.Context, string, *syncv1.PostflightRequest) (*syncv1.PostflightResponse, error)
}

type handler struct {
	tokenVerifier TokenVerifier
	service       Service
	logger        *slog.Logger
}

// RegisterSantaRoutes mounts Santa sync v1 endpoints on r.
func RegisterSantaRoutes(r chi.Router, tokenVerifier TokenVerifier, service Service, logger *slog.Logger) {
	h := handler{
		tokenVerifier: tokenVerifier,
		service:       service,
		logger:        logger,
	}
	r.Post("/api/santa/sync/preflight/{machine_id}", h.preflight)
	r.Post("/api/santa/sync/eventupload/{machine_id}", h.eventUpload)
	r.Post("/api/santa/sync/ruledownload/{machine_id}", h.ruleDownload)
	r.Post("/api/santa/sync/postflight/{machine_id}", h.postflight)
}

func (h handler) preflight(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(h, w, r, &syncv1.PreflightRequest{}, h.service.HandlePreflight)
}

func (h handler) eventUpload(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(h, w, r, &syncv1.EventUploadRequest{}, h.service.HandleEventUpload)
}

func (h handler) ruleDownload(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(h, w, r, &syncv1.RuleDownloadRequest{}, h.service.HandleRuleDownload)
}

func (h handler) postflight(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(h, w, r, &syncv1.PostflightRequest{}, h.service.HandlePostflight)
}

func handleSyncRequest[Req proto.Message, Resp proto.Message](
	h handler,
	w http.ResponseWriter,
	r *http.Request,
	req Req,
	handle func(context.Context, string, Req) (Resp, error),
) {
	if err := h.authorize(r); err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := validateTransportHeaders(r); err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := decodeRequest(r, req); err != nil {
		h.writeError(w, r, err)
		return
	}

	resp, err := handle(r.Context(), chi.URLParam(r, "machine_id"), req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := writeProtoResponse(w, resp); err != nil {
		h.log(r, http.StatusInternalServerError, err)
		writeStatusOnly(w, http.StatusInternalServerError)
	}
}

func (h handler) authorize(r *http.Request) error {
	authorization := r.Header.Get("Authorization")
	if !validBearerHeader(authorization) {
		return errUnauthorized
	}

	ok, err := h.tokenVerifier.VerifyBearerToken(r.Context(), authorization)
	if err != nil {
		return err
	}
	if !ok {
		return errUnauthorized
	}
	return nil
}

func validBearerHeader(authorization string) bool {
	scheme, value, ok := strings.Cut(authorization, " ")
	return ok && strings.EqualFold(scheme, "Bearer") && strings.TrimSpace(value) != "" &&
		!strings.Contains(strings.TrimSpace(value), " ")
}

func validateTransportHeaders(r *http.Request) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != protobufContentType {
		return errUnsupportedMedia
	}
	if !strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		return errUnsupportedMedia
	}
	return nil
}

func decodeRequest(r *http.Request, msg proto.Message) error {
	zr, err := gzip.NewReader(r.Body)
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidSyncBody, err)
	}
	defer zr.Close()

	payload, err := io.ReadAll(io.LimitReader(zr, maxRequestBodyBytes+1))
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidSyncBody, err)
	}
	if len(payload) > maxRequestBodyBytes {
		return errRequestBodyTooBig
	}
	if err := proto.Unmarshal(payload, msg); err != nil {
		return fmt.Errorf("%w: %w", errInvalidSyncBody, err)
	}
	return nil
}

func writeProtoResponse(w http.ResponseWriter, msg proto.Message) error {
	payload, err := marshalCompressedProto(msg)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", protobufContentType)
	w.Header().Set("Content-Encoding", "gzip")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(payload)
	return err
}

func marshalCompressedProto(msg proto.Message) ([]byte, error) {
	payload, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(payload); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (h handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := statusCodeForError(err)
	h.log(r, statusCode, err)
	writeStatusOnly(w, statusCode)
}

func statusCodeForError(err error) int {
	switch {
	case errors.Is(err, errUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, errUnsupportedMedia):
		return http.StatusUnsupportedMediaType
	case errors.Is(err, errInvalidSyncBody), errors.Is(err, errRequestBodyTooBig):
		return http.StatusBadRequest
	case errors.Is(err, santa.ErrNotImplemented):
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

func writeStatusOnly(w http.ResponseWriter, statusCode int) {
	w.Header().Del("Content-Type")
	w.Header().Del("Content-Encoding")
	w.WriteHeader(statusCode)
}

func (h handler) log(r *http.Request, statusCode int, err error) {
	if h.logger == nil {
		return
	}
	args := []any{
		"status", statusCode,
		"method", r.Method,
		"path", r.URL.Path,
		"machine_id", chi.URLParam(r, "machine_id"),
		"err", err,
	}
	if statusCode >= http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "santa sync request failed", args...)
		return
	}
	h.logger.WarnContext(r.Context(), "santa sync request rejected", args...)
}
