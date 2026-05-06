package osquery

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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
	return func(w http.ResponseWriter, r *http.Request) {
		var req coreosquery.ConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.Config(r.Context(), req.NodeKey)
		writeAgentResult(r, w, logger, "config", resp, err)
	}
}

func distributedReadHandler(svc *coreosquery.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req coreosquery.DistributedReadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.DistributedRead(r.Context(), req.NodeKey)
		writeAgentResult(r, w, logger, "distributed_read", resp, err)
	}
}

func distributedWriteHandler(svc *coreosquery.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req coreosquery.DistributedWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.DistributedWrite(r.Context(), req)
		writeAgentResult(r, w, logger, "distributed_write", resp, err)
	}
}

func logHandler(svc *coreosquery.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req coreosquery.LogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.Log(r.Context(), req.NodeKey)
		writeAgentResult(r, w, logger, "log", resp, err)
	}
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
			"osquery handler failed", "operation",
			operation,
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
