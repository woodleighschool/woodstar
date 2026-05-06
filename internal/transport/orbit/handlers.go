package orbit

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	coreorbit "github.com/woodleighschool/woodstar/internal/orbit"
)

const (
	capabilitiesHeader = "X-Fleet-Capabilities"
	orbitCapabilities  = "orbit_endpoints,end_user_email"
)

// RegisterRoutes mounts Orbit agent endpoints on r.
func RegisterRoutes(r chi.Router, svc *coreorbit.Service, logger *slog.Logger) {
	r.Post("/api/fleet/orbit/enroll", enrollHandler(svc, logger))
	r.Post("/api/fleet/orbit/config", configHandler(svc, logger))
	r.Put("/api/fleet/orbit/device_mapping", deviceMappingHandler(svc, logger))
	r.Head("/api/fleet/orbit/ping", pingHandler)
	registerStubs(r, svc)
}

func enrollHandler(svc *coreorbit.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req coreorbit.EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		host, nodeKey, err := svc.Enroll(r.Context(), req)
		switch {
		case errors.Is(err, coreorbit.ErrMissingHardwareUUID):
			writeAgentError(w, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, coreorbit.ErrInvalidEnrollSecret):
			// Do not reveal whether the secret was malformed vs unknown.
			logger.WarnContext(
				r.Context(),
				"orbit enroll rejected", "operation", "enroll",
				"reason", "invalid_enroll_secret",
				"hardware_uuid", req.HardwareUUID,
			)
			writeAgentError(w, http.StatusUnauthorized, "invalid enroll secret")
			return
		case err != nil:
			logger.ErrorContext(
				r.Context(),
				"orbit enroll failed", "operation", "enroll",
				"hardware_uuid", req.HardwareUUID,
				"err", err,
			)
			writeAgentError(w, http.StatusInternalServerError, "enrollment failed")
			return
		}

		logger.InfoContext(
			r.Context(),
			"orbit host enrolled", "operation", "enroll",
			"host_id", host.ID,
			"hardware_uuid", host.HardwareUUID,
			"display_name", host.DisplayName,
		)

		writeAgentJSON(w, http.StatusOK, coreorbit.EnrollResponse{OrbitNodeKey: nodeKey})
	}
}

func configHandler(svc *coreorbit.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req coreorbit.ConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.Config(r.Context(), req.OrbitNodeKey)
		if err != nil {
			logger.DebugContext(
				r.Context(),
				"orbit config rejected", "operation", "config",
				"reason", "invalid_node_key",
			)
			writeAgentError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		}
		writeAgentJSON(w, http.StatusOK, resp)
	}
}

func deviceMappingHandler(svc *coreorbit.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req coreorbit.DeviceMappingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := svc.SetDeviceMapping(r.Context(), req.OrbitNodeKey, req.Email); err != nil {
			logger.WarnContext(
				r.Context(),
				"orbit device mapping rejected", "operation", "device_mapping",
				"reason", "invalid_node_key",
			)
			writeAgentError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		}
		writeAgentJSON(w, http.StatusOK, map[string]any{})
	}
}

func pingHandler(w http.ResponseWriter, _ *http.Request) {
	writeOrbitHeaders(w)
	w.WriteHeader(http.StatusOK)
}

func writeAgentJSON(w http.ResponseWriter, status int, body any) {
	writeOrbitHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeOrbitHeaders(w http.ResponseWriter) {
	w.Header().Set(capabilitiesHeader, orbitCapabilities)
}

func writeAgentError(w http.ResponseWriter, status int, message string) {
	writeAgentJSON(w, status, errorResponse{Error: message})
}

type errorResponse struct {
	Error string `json:"error"`
}
