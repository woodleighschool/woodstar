// Package protocol exposes Orbit endpoints.
package protocol

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/httpx"
	"github.com/woodleighschool/woodstar/internal/orbit"
)

const (
	capabilitiesHeader     = "X-Fleet-Capabilities"
	orbitCapabilitiesValue = "orbit_endpoints,token_rotation,end_user_email"
	orbitRequestMaxBytes   = 1 << 20
)

// Server owns Orbit protocol routes.
type Server struct {
	service enrollmentService
	logger  *slog.Logger
}

type enrollmentService interface {
	Enroll(context.Context, orbit.EnrollRequest) (*hosts.Host, string, error)
	Config(context.Context, string) (orbit.ConfigResponse, error)
	SetPrimaryUser(context.Context, string, string) error
	SetDeviceAuthToken(context.Context, string, string) error
	ValidateDeviceAuthToken(context.Context, string) error
}

// NewServer returns an Orbit protocol server.
func NewServer(service enrollmentService, logger *slog.Logger) *Server {
	return &Server{service: service, logger: logger}
}

// RegisterRoutes mounts Orbit endpoints on r.
func (s *Server) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(orbitCapabilities)
		r.Post("/api/fleet/orbit/enroll", orbitEnrollHandler(s.service, s.logger))
		r.Post("/api/fleet/orbit/config", orbitConfigHandler(s.service, s.logger))
		r.Put("/api/fleet/orbit/device_mapping", orbitDeviceMappingHandler(s.service, s.logger))
		r.Post("/api/fleet/orbit/device_token", orbitDeviceTokenHandler(s.service, s.logger))
		r.Head("/api/fleet/orbit/ping", orbitPingHandler)
		r.Head("/api/latest/fleet/device/{token}/ping", orbitDevicePingHandler(s.service, s.logger))
	})
}

func orbitEnrollHandler(svc enrollmentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[orbit.EnrollRequest](w, r, orbitRequestMaxBytes)
		if err != nil {
			httpx.WriteDecodeError(w, err)
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

func orbitConfigHandler(svc enrollmentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[orbit.ConfigRequest](w, r, orbitRequestMaxBytes)
		if err != nil {
			httpx.WriteDecodeError(w, err)
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

func orbitDeviceMappingHandler(svc enrollmentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[orbit.DeviceMappingRequest](w, r, orbitRequestMaxBytes)
		if err != nil {
			httpx.WriteDecodeError(w, err)
			return
		}
		err = svc.SetPrimaryUser(r.Context(), req.OrbitNodeKey, req.Email)
		switch {
		case errors.Is(err, dbutil.ErrInvalidInput):
			httpx.WriteError(w, http.StatusBadRequest, "invalid primary user email")
			return
		case errors.Is(err, dbutil.ErrNotFound):
			logger.WarnContext(
				r.Context(),
				"orbit device mapping rejected", "operation", "device_mapping",
				"reason", "invalid_node_key",
			)
			httpx.WriteError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		case err != nil:
			logger.ErrorContext(
				r.Context(), "orbit device mapping failed",
				"operation", "device_mapping", "err", err,
			)
			httpx.WriteError(w, http.StatusInternalServerError, "device mapping failed")
			return
		}
		httpx.Write(w, http.StatusOK, struct{}{})
	}
}

func orbitPingHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func orbitDeviceTokenHandler(svc enrollmentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpx.Decode[orbit.DeviceTokenRequest](w, r, orbitRequestMaxBytes)
		if err != nil {
			httpx.WriteDecodeError(w, err)
			return
		}
		err = svc.SetDeviceAuthToken(r.Context(), req.OrbitNodeKey, req.DeviceAuthToken)
		switch {
		case errors.Is(err, orbit.ErrInvalidDeviceAuthToken):
			httpx.WriteError(w, http.StatusBadRequest, "invalid device auth token")
		case errors.Is(err, dbutil.ErrNotFound):
			httpx.WriteError(w, http.StatusUnauthorized, "invalid orbit node key")
		case errors.Is(err, dbutil.ErrAlreadyExists):
			httpx.WriteError(w, http.StatusConflict, "device auth token already exists")
		case err != nil:
			logger.ErrorContext(
				r.Context(), "rotate Orbit device token failed",
				"operation", "device_token", "err", err,
			)
			httpx.WriteError(w, http.StatusInternalServerError, "device token rotation failed")
		default:
			httpx.Write(w, http.StatusOK, struct{}{})
		}
	}
}

func orbitDevicePingHandler(svc enrollmentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := svc.ValidateDeviceAuthToken(r.Context(), chi.URLParam(r, "token"))
		switch {
		case errors.Is(err, dbutil.ErrNotFound):
			httpx.WriteError(w, http.StatusUnauthorized, "invalid device auth token")
		case err != nil:
			logger.ErrorContext(
				r.Context(), "validate Orbit device token failed",
				"operation", "device_ping", "err", err,
			)
			httpx.WriteError(w, http.StatusInternalServerError, "device token validation failed")
		default:
			w.WriteHeader(http.StatusOK)
		}
	}
}

func orbitCapabilities(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(capabilitiesHeader, orbitCapabilitiesValue)
		next.ServeHTTP(w, r)
	})
}
