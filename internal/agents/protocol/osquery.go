package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agents"
	"github.com/woodleighschool/woodstar/internal/agents/osquery"
)

// RegisterOsqueryRoutes mounts osquery TLS-plugin endpoints on r.
func RegisterOsqueryRoutes(r chi.Router, svc *osquery.Service, logger *slog.Logger) {
	for _, prefix := range []string{"/api/osquery", "/api/v1/osquery"} {
		r.Post(prefix+"/enroll", osqueryEnrollHandler(svc, logger))
		r.Post(prefix+"/config", osqueryConfigHandler(svc, logger))
		r.Post(prefix+"/distributed/read", osqueryDistributedReadHandler(svc, logger))
		r.Post(prefix+"/distributed/write", osqueryDistributedWriteHandler(svc, logger))
		r.Post(prefix+"/log", osqueryLogHandler(svc, logger))
		r.Post(prefix+"/carve/begin", osqueryEmptyJSONHandler)
		r.Post(prefix+"/carve/block", osqueryEmptyJSONHandler)
	}
}

func osqueryEnrollHandler(svc *osquery.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req osquery.EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		nodeKey, err := svc.Enroll(r.Context(), req)
		switch {
		case errors.Is(err, agents.ErrInvalidEnrollSecret):
			logger.WarnContext(
				r.Context(),
				"osquery enroll rejected", "operation", "enroll",
				"reason", "invalid_enroll_secret",
				"host_identifier", req.HostIdentifier,
			)
			writeError(w, http.StatusUnauthorized, "invalid enroll secret")
			return
		case errors.Is(err, agents.ErrMissingHardwareUUID):
			writeError(w, http.StatusBadRequest, err.Error())
			return
		case err != nil:
			logger.ErrorContext(
				r.Context(),
				"osquery enroll failed", "operation", "enroll",
				"host_identifier", req.HostIdentifier,
				"err", err,
			)
			writeError(w, http.StatusInternalServerError, "enrollment failed")
			return
		}
		writeJSON(w, http.StatusOK, osquery.EnrollResponse{NodeKey: nodeKey})
	}
}

func osqueryConfigHandler(svc *osquery.Service, logger *slog.Logger) http.HandlerFunc {
	return osqueryNodeKeyHandler(logger, "config",
		func(ctx context.Context, req osquery.ConfigRequest, publicIP string) (any, error) {
			return svc.Config(ctx, req.NodeKey, publicIP)
		},
	)
}

func osqueryDistributedReadHandler(svc *osquery.Service, logger *slog.Logger) http.HandlerFunc {
	return osqueryNodeKeyHandler(logger, "distributed_read",
		func(ctx context.Context, req osquery.DistributedReadRequest, publicIP string) (any, error) {
			return svc.DistributedRead(ctx, req.NodeKey, publicIP)
		},
	)
}

func osqueryDistributedWriteHandler(svc *osquery.Service, logger *slog.Logger) http.HandlerFunc {
	return osqueryNodeKeyHandler(logger, "distributed_write",
		func(ctx context.Context, req osquery.DistributedWriteRequest, publicIP string) (any, error) {
			return svc.DistributedWrite(ctx, req, publicIP)
		},
	)
}

func osqueryLogHandler(svc *osquery.Service, logger *slog.Logger) http.HandlerFunc {
	return osqueryNodeKeyHandler(logger, "log",
		func(ctx context.Context, req osquery.LogRequest, publicIP string) (any, error) {
			return svc.Log(ctx, req.NodeKey, publicIP, req)
		},
	)
}

func osqueryNodeKeyHandler[T any](
	logger *slog.Logger,
	operation string,
	handle func(context.Context, T, string) (any, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := handle(r.Context(), req, clientIP(r))
		writeOsqueryResult(r, w, logger, operation, resp, err)
	}
}

func clientIP(r *http.Request) string {
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
		logger.ErrorContext(
			r.Context(),
			"osquery handler failed",
			"operation", operation,
			"err",
			err,
		)
		writeError(w, http.StatusInternalServerError, "request failed")
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func osqueryEmptyJSONHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, struct{}{})
}
