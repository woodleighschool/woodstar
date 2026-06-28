// Package protocol exposes Orbit endpoints.
package protocol

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/httpx"
	"github.com/woodleighschool/woodstar/internal/orbit"
)

const (
	capabilitiesHeader     = "X-Fleet-Capabilities"
	orbitCapabilitiesValue = "orbit_endpoints,end_user_email"
)

// Server owns Orbit protocol routes.
type Server struct {
	service *orbit.EnrollmentService
	logger  *slog.Logger
}

// NewServer returns an Orbit protocol server.
func NewServer(service *orbit.EnrollmentService, logger *slog.Logger) *Server {
	return &Server{service: service, logger: logger}
}

// RegisterRoutes mounts Orbit endpoints on r.
func (s *Server) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(orbitCapabilities)
		r.Post("/api/fleet/orbit/enroll", orbitEnrollHandler(s.service, s.logger))
		r.Post("/api/fleet/orbit/config", orbitConfigHandler(s.service, s.logger))
		r.Put("/api/fleet/orbit/device_mapping", orbitDeviceMappingHandler(s.service, s.logger))
		r.Head("/api/fleet/orbit/ping", orbitPingHandler)
		registerOrbitCompatibilityRoutes(r, s.service)
	})
}

func orbitEnrollHandler(svc *orbit.EnrollmentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[orbit.EnrollRequest](r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		host, nodeKey, err := svc.Enroll(r.Context(), req)
		switch {
		case errors.Is(err, enrollment.ErrMissingHardwareUUID):
			httpx.WriteError(w, http.StatusBadRequest, "hardware_uuid is required")
			return
		case errors.Is(err, agentauth.ErrInvalidSecret):
			logger.WarnContext(
				r.Context(),
				"orbit enroll rejected", "operation", "enroll",
				"reason", "invalid_enroll_secret",
				"hardware_uuid", req.HardwareUUID,
			)
			httpx.WriteError(w, http.StatusUnauthorized, "invalid enroll secret")
			return
		case err != nil:
			logger.ErrorContext(
				r.Context(),
				"orbit enroll failed", "operation", "enroll",
				"hardware_uuid", req.HardwareUUID,
				"err", err,
			)
			httpx.WriteError(w, http.StatusInternalServerError, "enrollment failed")
			return
		}

		logger.DebugContext(
			r.Context(),
			"orbit host enrolled", "operation", "enroll",
			"host_id", host.ID,
			"hardware_uuid", host.Hardware.UUID,
			"display_name", host.DisplayName,
		)
		httpx.Write(w, http.StatusOK, orbit.EnrollResponse{OrbitNodeKey: nodeKey})
	}
}

func orbitConfigHandler(svc *orbit.EnrollmentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[orbit.ConfigRequest](r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resp, err := svc.Config(r.Context(), req.OrbitNodeKey)
		if err != nil {
			logger.DebugContext(
				r.Context(),
				"orbit config rejected", "operation", "config",
				"reason", "invalid_node_key",
			)
			httpx.WriteError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		}
		httpx.Write(w, http.StatusOK, resp)
	}
}

func orbitDeviceMappingHandler(svc *orbit.EnrollmentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[orbit.DeviceMappingRequest](r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := svc.SetPrimaryUser(r.Context(), req.OrbitNodeKey, req.Email); err != nil {
			logger.WarnContext(
				r.Context(),
				"orbit device mapping rejected", "operation", "device_mapping",
				"reason", "invalid_node_key",
			)
			httpx.WriteError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		}
		httpx.Write(w, http.StatusOK, struct{}{})
	}
}

func orbitPingHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func orbitCapabilities(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(capabilitiesHeader, orbitCapabilitiesValue)
		next.ServeHTTP(w, r)
	})
}
