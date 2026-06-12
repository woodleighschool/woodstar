// Package protocol exposes osquery TLS-plugin endpoints.
package protocol

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/httpjson"
	"github.com/woodleighschool/woodstar/internal/osquery"
)

var osqueryEmptyJSONPaths = []string{"/carve/begin", "/carve/block"}

// RegisterOsqueryRoutes mounts osquery TLS-plugin endpoints on r.
func RegisterOsqueryRoutes(r chi.Router, svc *osquery.AgentService, logger *slog.Logger) {
	for _, prefix := range []string{"/api/osquery", "/api/v1/osquery"} {
		r.Post(prefix+"/enroll", osqueryEnrollHandler(svc, logger))
		r.Post(prefix+"/config", osqueryConfigHandler(svc, logger))
		r.Post(prefix+"/distributed/read", osqueryDistributedReadHandler(svc, logger))
		r.Post(prefix+"/distributed/write", osqueryDistributedWriteHandler(svc, logger))
		r.Post(prefix+"/log", osqueryLogHandler(svc, logger))
		for _, path := range osqueryEmptyJSONPaths {
			r.Post(prefix+path, osqueryEmptyJSONHandler)
		}
	}
}

func osqueryEnrollHandler(svc *osquery.AgentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpjson.Decode[osquery.EnrollRequest](r)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid request body")
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
			httpjson.WriteError(w, http.StatusUnauthorized, "invalid enroll secret")
			return
		case errors.Is(err, enrollment.ErrMissingHardwareUUID):
			httpjson.WriteError(w, http.StatusBadRequest, "hardware_uuid is required")
			return
		case err != nil:
			logger.ErrorContext(
				r.Context(),
				"osquery enroll failed", "operation", "enroll",
				"host_identifier", req.HostIdentifier,
				"err", err,
			)
			httpjson.WriteError(w, http.StatusInternalServerError, "enrollment failed")
			return
		}
		httpjson.Write(w, http.StatusOK, osquery.EnrollResponse{NodeKey: nodeKey})
	}
}

func osqueryConfigHandler(svc *osquery.AgentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpjson.Decode[osquery.ConfigRequest](r)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.Config(r.Context(), req.NodeKey, clientIP(r))
		writeOsqueryResult(r, w, logger, "config", resp, err)
	}
}

func osqueryDistributedReadHandler(svc *osquery.AgentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpjson.Decode[osquery.DistributedReadRequest](r)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.DistributedRead(r.Context(), req.NodeKey, clientIP(r))
		writeOsqueryResult(r, w, logger, "distributed_read", resp, err)
	}
}

func osqueryDistributedWriteHandler(svc *osquery.AgentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpjson.Decode[osquery.DistributedWriteRequest](r)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.DistributedWrite(r.Context(), req, clientIP(r))
		writeOsqueryResult(r, w, logger, "distributed_write", resp, err)
	}
}

func osqueryLogHandler(svc *osquery.AgentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpjson.Decode[osquery.LogRequest](r)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.Log(r.Context(), req.NodeKey, clientIP(r), req)
		writeOsqueryResult(r, w, logger, "log", resp, err)
	}
}

func clientIP(r *http.Request) string {
	if ip := chimiddleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return parseIP(host)
	}
	return parseIP(r.RemoteAddr)
}

func parseIP(value string) string {
	ip, err := netip.ParseAddr(value)
	if err != nil {
		return ""
	}
	return ip.String()
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
		httpjson.WriteError(w, http.StatusInternalServerError, "request failed")
		return
	}
	httpjson.Write(w, http.StatusOK, body)
}

func osqueryEmptyJSONHandler(w http.ResponseWriter, _ *http.Request) {
	httpjson.Write(w, http.StatusOK, struct{}{})
}
