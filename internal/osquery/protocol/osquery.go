// Package protocol exposes osquery TLS-plugin endpoints.
package protocol

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/httpx"
	"github.com/woodleighschool/woodstar/internal/osquery"
)

const (
	osqueryPath                     = "/api/v1/osquery"
	osqueryRequestMaxBytes          = 1 << 20
	osqueryDistributedWriteMaxBytes = 5 << 20
	osqueryLogMaxBytes              = 10 << 20
)

// Server owns osquery TLS-plugin routes.
type Server struct {
	service *osquery.AgentService
	logger  *slog.Logger
}

// NewServer returns an osquery protocol server.
func NewServer(service *osquery.AgentService, logger *slog.Logger) *Server {
	return &Server{service: service, logger: logger}
}

// RegisterRoutes mounts osquery TLS-plugin endpoints on r.
func (s *Server) RegisterRoutes(r chi.Router) {
	r.Post(osqueryPath+"/enroll", osqueryEnrollHandler(s.service, s.logger))
	r.Post(osqueryPath+"/config", osqueryConfigHandler(s.service, s.logger))
	r.Post(osqueryPath+"/distributed/read", osqueryDistributedReadHandler(s.service, s.logger))
	r.Post(osqueryPath+"/distributed/write", osqueryDistributedWriteHandler(s.service, s.logger))
	r.Post(osqueryPath+"/log", osqueryLogHandler(s.service, s.logger))
}

func osqueryEnrollHandler(svc *osquery.AgentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[osquery.EnrollRequest](w, r, osqueryRequestMaxBytes)
		if err != nil {
			httpx.WriteDecodeError(w, err)
			return
		}
		nodeKey, err := svc.Enroll(r.Context(), req)
		switch {
		case errors.Is(err, agentauth.ErrInvalidSecret):
			logger.WarnContext(
				r.Context(),
				"osquery enroll rejected", "operation", "enroll",
				"reason", "invalid_enroll_secret",
				"host_identifier", req.HostIdentifier,
			)
			httpx.WriteError(w, http.StatusUnauthorized, "invalid enroll secret")
			return
		case errors.Is(err, enrollment.ErrMissingHardwareUUID):
			httpx.WriteError(w, http.StatusBadRequest, "hardware_uuid is required")
			return
		case err != nil:
			logger.ErrorContext(
				r.Context(),
				"osquery enroll failed", "operation", "enroll",
				"host_identifier", req.HostIdentifier,
				"err", err,
			)
			httpx.WriteError(w, http.StatusInternalServerError, "enrollment failed")
			return
		}
		httpx.Write(w, http.StatusOK, osquery.EnrollResponse{NodeKey: nodeKey})
	}
}

func osqueryConfigHandler(svc *osquery.AgentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[osquery.ConfigRequest](w, r, osqueryRequestMaxBytes)
		if err != nil {
			httpx.WriteDecodeError(w, err)
			return
		}
		resp, err := svc.Config(r.Context(), req.NodeKey, chimiddleware.GetClientIP(r.Context()))
		writeOsqueryResult(r, w, logger, "config", resp, err)
	}
}

func osqueryDistributedReadHandler(svc *osquery.AgentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[osquery.DistributedReadRequest](w, r, osqueryRequestMaxBytes)
		if err != nil {
			httpx.WriteDecodeError(w, err)
			return
		}
		resp, err := svc.DistributedRead(r.Context(), req.NodeKey, chimiddleware.GetClientIP(r.Context()))
		writeOsqueryResult(r, w, logger, "distributed_read", resp, err)
	}
}

func osqueryDistributedWriteHandler(svc *osquery.AgentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[osquery.DistributedWriteRequest](w, r, osqueryDistributedWriteMaxBytes)
		if err != nil {
			httpx.WriteDecodeError(w, err)
			return
		}
		resp, err := svc.DistributedWrite(r.Context(), req, chimiddleware.GetClientIP(r.Context()))
		writeOsqueryResult(r, w, logger, "distributed_write", resp, err)
	}
}

func osqueryLogHandler(svc *osquery.AgentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[osquery.LogRequest](w, r, osqueryLogMaxBytes)
		if err != nil {
			httpx.WriteDecodeError(w, err)
			return
		}
		resp, err := svc.Log(r.Context(), req.NodeKey, chimiddleware.GetClientIP(r.Context()), req)
		writeOsqueryResult(r, w, logger, "log", resp, err)
	}
}

func writeOsqueryResult(
	r *http.Request,
	w http.ResponseWriter,
	logger *slog.Logger,
	operation string,
	body any,
	err error,
) {
	if err != nil {
		logger.ErrorContext(r.Context(), "osquery handler failed", "operation", operation, "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "request failed")
		return
	}
	httpx.Write(w, http.StatusOK, body)
}
