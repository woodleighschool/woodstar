// Package protocol exposes Santa sync endpoints.
package protocol

import (
	"context"
	"log/slog"
	"net/http"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/proto"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/santa"
)

type machineIDProtoMessage interface {
	proto.Message
	GetMachineId() string
}

// SyncService handles decoded Santa sync requests.
type SyncService interface {
	Preflight(ctx context.Context, machineID string, req santa.PreflightRequest) (santa.PreflightResponse, error)
	EventUpload(ctx context.Context, machineID string, req santa.EventUploadRequest) (santa.EventUploadResponse, error)
	RuleDownload(
		ctx context.Context,
		machineID string,
		req santa.RuleDownloadRequest,
	) (santa.RuleDownloadResponse, error)
	Postflight(ctx context.Context, machineID string, req santa.PostflightRequest) (santa.PostflightResponse, error)
}

type handler struct {
	secretVerifier agentauth.SecretVerifier
	service        SyncService
	logger         *slog.Logger
}

// Server owns Santa sync protocol routes.
type Server struct {
	secretVerifier agentauth.SecretVerifier
	service        SyncService
	logger         *slog.Logger
}

// NewServer returns a Santa sync protocol server.
func NewServer(secretVerifier agentauth.SecretVerifier, service SyncService, logger *slog.Logger) *Server {
	return &Server{secretVerifier: secretVerifier, service: service, logger: logger}
}

// RegisterRoutes mounts Santa sync v1 endpoints on r.
func (s *Server) RegisterRoutes(r chi.Router) {
	h := handler{
		secretVerifier: s.secretVerifier,
		service:        s.service,
		logger:         s.logger,
	}
	r.Post("/santa/sync/preflight/{machine_id}", h.preflight)
	r.Post("/santa/sync/eventupload/{machine_id}", h.eventUpload)
	r.Post("/santa/sync/ruledownload/{machine_id}", h.ruleDownload)
	r.Post("/santa/sync/postflight/{machine_id}", h.postflight)
}

func (h handler) preflight(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(
		h,
		w,
		r,
		&syncv1.PreflightRequest{},
		preflightRequestFromProto,
		h.service.Preflight,
		preflightResponseToProto,
	)
}

func (h handler) eventUpload(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(
		h,
		w,
		r,
		&syncv1.EventUploadRequest{},
		eventUploadRequestFromProto,
		h.service.EventUpload,
		eventUploadResponseToProto,
	)
}

func (h handler) ruleDownload(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(
		h,
		w,
		r,
		&syncv1.RuleDownloadRequest{},
		ruleDownloadRequestFromProto,
		h.service.RuleDownload,
		ruleDownloadResponseToProto,
	)
}

func (h handler) postflight(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(
		h,
		w,
		r,
		&syncv1.PostflightRequest{},
		postflightRequestFromProto,
		h.service.Postflight,
		postflightResponseToProto,
	)
}

func handleSyncRequest[ProtoReq machineIDProtoMessage, DomainReq any, DomainResp any, ProtoResp proto.Message](
	h handler,
	w http.ResponseWriter,
	r *http.Request,
	req ProtoReq,
	fromProto func(ProtoReq) (DomainReq, error),
	handle func(context.Context, string, DomainReq) (DomainResp, error),
	toProto func(DomainResp) (ProtoResp, error),
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
	machineID := chi.URLParam(r, "machine_id")
	if err := validateRequestMachineID(machineID, req); err != nil {
		h.writeError(w, r, err)
		return
	}
	domainReq, err := fromProto(req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	resp, err := handle(r.Context(), machineID, domainReq)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	protoResp, err := toProto(resp)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := writeProtoResponse(w, protoResp); err != nil {
		h.log(r, http.StatusInternalServerError, err)
		writeStatusOnly(w, http.StatusInternalServerError)
	}
}
