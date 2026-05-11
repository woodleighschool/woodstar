// Package agentapi exposes Orbit and osquery HTTP edges and agent-facing Huma
// handlers (queries, checks, live queries).
package agentapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agents"
	"github.com/woodleighschool/woodstar/internal/agents/orbit"
)

const (
	capabilitiesHeader = "X-Fleet-Capabilities"
	orbitCapabilities  = "orbit_endpoints,end_user_email"
)

// RegisterOrbitRoutes mounts Orbit agent endpoints on r.
func RegisterOrbitRoutes(r chi.Router, svc *orbit.Service, logger *slog.Logger) {
	r.Post("/api/fleet/orbit/enroll", orbitEnrollHandler(svc, logger))
	r.Post("/api/fleet/orbit/config", orbitConfigHandler(svc, logger))
	r.Put("/api/fleet/orbit/device_mapping", orbitDeviceMappingHandler(svc, logger))
	r.Head("/api/fleet/orbit/ping", orbitPingHandler)
	registerOrbitStubs(r, svc)
}

func orbitEnrollHandler(svc *orbit.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req orbit.EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeOrbitError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		host, nodeKey, err := svc.Enroll(r.Context(), req)
		switch {
		case errors.Is(err, agents.ErrMissingHardwareUUID):
			writeOrbitError(w, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, agents.ErrInvalidEnrollSecret):
			logger.WarnContext(
				r.Context(),
				"orbit enroll rejected", "operation", "enroll",
				"reason", "invalid_enroll_secret",
				"hardware_uuid", req.HardwareUUID,
			)
			writeOrbitError(w, http.StatusUnauthorized, "invalid enroll secret")
			return
		case err != nil:
			logger.ErrorContext(
				r.Context(),
				"orbit enroll failed", "operation", "enroll",
				"hardware_uuid", req.HardwareUUID,
				"err", err,
			)
			writeOrbitError(w, http.StatusInternalServerError, "enrollment failed")
			return
		}

		logger.InfoContext(
			r.Context(),
			"orbit host enrolled", "operation", "enroll",
			"host_id", host.ID,
			"hardware_uuid", host.HardwareUUID,
			"display_name", host.DisplayName,
		)

		writeOrbitJSON(w, http.StatusOK, orbit.EnrollResponse{OrbitNodeKey: nodeKey})
	}
}

func orbitConfigHandler(svc *orbit.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req orbit.ConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeOrbitError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.Config(r.Context(), req.OrbitNodeKey)
		if err != nil {
			logger.DebugContext(
				r.Context(),
				"orbit config rejected", "operation", "config",
				"reason", "invalid_node_key",
			)
			writeOrbitError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		}
		writeOrbitJSON(w, http.StatusOK, resp)
	}
}

func orbitDeviceMappingHandler(svc *orbit.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req orbit.DeviceMappingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeOrbitError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := svc.SetDeviceMapping(r.Context(), req.OrbitNodeKey, req.Email); err != nil {
			logger.WarnContext(
				r.Context(),
				"orbit device mapping rejected", "operation", "device_mapping",
				"reason", "invalid_node_key",
			)
			writeOrbitError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		}
		writeOrbitJSON(w, http.StatusOK, struct{}{})
	}
}

func orbitPingHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(capabilitiesHeader, orbitCapabilities)
	w.WriteHeader(http.StatusOK)
}

func writeOrbitJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set(capabilitiesHeader, orbitCapabilities)
	writeJSON(w, status, body)
}

func writeOrbitError(w http.ResponseWriter, status int, message string) {
	w.Header().Set(capabilitiesHeader, orbitCapabilities)
	writeError(w, status, message)
}

// ----- stubs -----

func registerOrbitStubs(r chi.Router, svc *orbit.Service) {
	r.Post("/api/fleet/orbit/scripts/request", requireOrbitNodeKey(svc, func(w http.ResponseWriter, _ *http.Request) {
		writeOrbitJSON(w, http.StatusOK, scriptsRequestResponse{Scripts: []string{}})
	}))
	r.Post("/api/fleet/orbit/scripts/result", requireOrbitNodeKey(svc, orbitNoContentHandler))
	r.Post("/api/fleet/orbit/software_install/details", requireOrbitNodeKey(svc, orbitEmptyObjectHandler))
	r.Post("/api/fleet/orbit/software_install/package", requireOrbitNodeKey(svc, orbitEmptyObjectHandler))
	r.Post("/api/fleet/orbit/software_install/result", requireOrbitNodeKey(svc, orbitNoContentHandler))
	r.Post("/api/fleet/orbit/setup_experience/init", requireOrbitNodeKey(svc, orbitEmptyObjectHandler))
	r.Post("/api/fleet/orbit/setup_experience/status", requireOrbitNodeKey(svc, setupExperienceStatusHandler))
	r.Post("/api/fleet/orbit/device_token", requireOrbitNodeKey(svc, orbitNoContentHandler))
	r.Post("/api/fleet/orbit/disk_encryption_key", requireOrbitNodeKey(svc, orbitNoContentHandler))
	r.Post("/api/fleet/orbit/luks_data", requireOrbitNodeKey(svc, orbitNoContentHandler))
}

func requireOrbitNodeKey(svc *orbit.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OrbitNodeKey string `json:"orbit_node_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeOrbitError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := svc.ValidateNodeKey(r.Context(), req.OrbitNodeKey); err != nil {
			writeOrbitError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		}
		next(w, r)
	}
}

type scriptsRequestResponse struct {
	Scripts []string `json:"scripts"`
}

type setupExperienceResponse struct {
	SetupExperienceResults *struct{} `json:"setup_experience_results"`
	OK                     bool      `json:"ok"`
}

func orbitEmptyObjectHandler(w http.ResponseWriter, _ *http.Request) {
	writeOrbitJSON(w, http.StatusOK, struct{}{})
}

func setupExperienceStatusHandler(w http.ResponseWriter, _ *http.Request) {
	writeOrbitJSON(w, http.StatusOK, setupExperienceResponse{OK: true})
}

func orbitNoContentHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(capabilitiesHeader, orbitCapabilities)
	w.WriteHeader(http.StatusNoContent)
}
