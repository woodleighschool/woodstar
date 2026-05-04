package osquery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// RegisterRoutes mounts osquery TLS-plugin endpoints on r.
func RegisterRoutes(r chi.Router, svc *Service) {
	for _, prefix := range []string{"/api/osquery", "/api/v1/osquery"} {
		r.Post(prefix+"/enroll", enrollHandler(svc))
		r.Post(prefix+"/config", configHandler(svc))
		r.Post(prefix+"/distributed/read", distributedReadHandler(svc))
		r.Post(prefix+"/distributed/write", distributedWriteHandler(svc))
		r.Post(prefix+"/log", logHandler(svc))
		r.Post(prefix+"/carve/begin", emptyJSONHandler)
		r.Post(prefix+"/carve/block", emptyJSONHandler)
	}
}

func enrollHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		nodeKey, err := svc.Enroll(r.Context(), req)
		switch {
		case errors.Is(err, ErrInvalidEnrollSecret):
			writeAgentError(w, http.StatusUnauthorized, "invalid enroll secret")
			return
		case errors.Is(err, ErrMissingHardwareUUID):
			writeAgentError(w, http.StatusBadRequest, err.Error())
			return
		case err != nil:
			log.Error().Err(err).Str("host_identifier", req.HostIdentifier).Msg("osquery enroll failed")
			writeAgentError(w, http.StatusInternalServerError, "enrollment failed")
			return
		}
		writeAgentJSON(w, http.StatusOK, EnrollResponse{NodeKey: nodeKey})
	}
}

func configHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.Config(r.Context(), req.NodeKey)
		writeAgentResult(w, resp, err)
	}
}

func distributedReadHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DistributedReadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.DistributedRead(r.Context(), req.NodeKey)
		writeAgentResult(w, resp, err)
	}
}

func distributedWriteHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DistributedWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.DistributedWrite(r.Context(), req)
		writeAgentResult(w, resp, err)
	}
}

func logHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.Log(r.Context(), req.NodeKey)
		writeAgentResult(w, resp, err)
	}
}

func writeAgentResult(w http.ResponseWriter, body any, err error) {
	if err != nil {
		log.Error().Err(err).Msg("osquery handler failed")
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

func emptyJSONHandler(w http.ResponseWriter, _ *http.Request) {
	writeAgentJSON(w, http.StatusOK, map[string]any{})
}
