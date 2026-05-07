package osquery

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"

	"github.com/go-chi/chi/v5"

	coreosquery "github.com/woodleighschool/woodstar/internal/osquery"
)

// RegisterRoutes mounts osquery TLS-plugin endpoints on r.
func RegisterRoutes(r chi.Router, svc *coreosquery.Service, logger *slog.Logger) {
	for _, prefix := range []string{"/api/osquery", "/api/v1/osquery"} {
		r.Post(prefix+"/enroll", enrollHandler(svc, logger))
		r.Post(prefix+"/config", configHandler(svc, logger))
		r.Post(prefix+"/distributed/read", distributedReadHandler(svc, logger))
		r.Post(prefix+"/distributed/write", distributedWriteHandler(svc, logger))
		r.Post(prefix+"/log", logHandler(svc, logger))
		r.Post(prefix+"/carve/begin", emptyJSONHandler)
		r.Post(prefix+"/carve/block", emptyJSONHandler)
	}
}

func enrollHandler(svc *coreosquery.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req coreosquery.EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		nodeKey, err := svc.Enroll(r.Context(), req)
		switch {
		case errors.Is(err, coreosquery.ErrInvalidEnrollSecret):
			logger.WarnContext(
				r.Context(),
				"osquery enroll rejected", "operation", "enroll",
				"reason", "invalid_enroll_secret",
				"host_identifier", req.HostIdentifier,
			)
			writeAgentError(w, http.StatusUnauthorized, "invalid enroll secret")
			return
		case errors.Is(err, coreosquery.ErrMissingHardwareUUID):
			writeAgentError(w, http.StatusBadRequest, err.Error())
			return
		case err != nil:
			logger.ErrorContext(
				r.Context(),
				"osquery enroll failed", "operation", "enroll",
				"host_identifier", req.HostIdentifier,
				"err", err,
			)
			writeAgentError(w, http.StatusInternalServerError, "enrollment failed")
			return
		}
		writeAgentJSON(w, http.StatusOK, coreosquery.EnrollResponse{NodeKey: nodeKey})
	}
}

func configHandler(svc *coreosquery.Service, logger *slog.Logger) http.HandlerFunc {
	return nodeKeyHandler(logger, "config",
		func(ctx context.Context, req coreosquery.ConfigRequest, publicIP string) (any, error) {
			return svc.Config(ctx, req.NodeKey, publicIP)
		},
	)
}

func distributedReadHandler(svc *coreosquery.Service, logger *slog.Logger) http.HandlerFunc {
	return nodeKeyHandler(logger, "distributed_read",
		func(ctx context.Context, req coreosquery.DistributedReadRequest, publicIP string) (any, error) {
			return svc.DistributedRead(ctx, req.NodeKey, publicIP)
		},
	)
}

func distributedWriteHandler(svc *coreosquery.Service, logger *slog.Logger) http.HandlerFunc {
	return nodeKeyHandler(logger, "distributed_write",
		func(ctx context.Context, req coreosquery.DistributedWriteRequest, publicIP string) (any, error) {
			return svc.DistributedWrite(ctx, req, publicIP)
		},
	)
}

func logHandler(svc *coreosquery.Service, logger *slog.Logger) http.HandlerFunc {
	return nodeKeyHandler(logger, "log",
		func(ctx context.Context, req coreosquery.LogRequest, publicIP string) (any, error) {
			return svc.Log(ctx, req.NodeKey, publicIP)
		},
	)
}

func nodeKeyHandler[T any](
	logger *slog.Logger,
	operation string,
	handle func(context.Context, T, string) (any, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := handle(r.Context(), req, clientIP(r))
		writeAgentResult(r, w, logger, operation, resp, err)
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

func writeAgentResult(
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
		writeAgentError(w, http.StatusInternalServerError, "request failed")
		return
	}
	writeAgentJSON(w, http.StatusOK, body)
}

func writeAgentJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeAgentError(w http.ResponseWriter, status int, message string) {
	writeAgentJSON(w, status, errorResponse{Error: message})
}

type errorResponse struct {
	Error string `json:"error"`
}

func emptyJSONHandler(w http.ResponseWriter, _ *http.Request) {
	writeAgentJSON(w, http.StatusOK, map[string]any{})
}
