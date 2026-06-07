// Package protocol exposes Orbit endpoints.
package protocol

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/httpjson"
	"github.com/woodleighschool/woodstar/internal/orbit"
)

const (
	capabilitiesHeader     = "X-Fleet-Capabilities"
	orbitCapabilitiesValue = "orbit_endpoints,end_user_email"
)

// RegisterOrbitRoutes mounts Orbit endpoints on r.
func RegisterOrbitRoutes(r chi.Router, svc *orbit.EnrollmentService, logger *slog.Logger) {
	r.Group(func(r chi.Router) {
		r.Use(orbitCapabilities)
		r.Post("/api/fleet/orbit/enroll", orbitEnrollHandler(svc, logger))
		r.Post("/api/fleet/orbit/config", orbitConfigHandler(svc, logger))
		r.Put("/api/fleet/orbit/device_mapping", orbitDeviceMappingHandler(svc, logger))
		r.Head("/api/fleet/orbit/ping", orbitPingHandler)
		registerOrbitCompatibilityRoutes(r, svc, logger)
	})
}

func orbitEnrollHandler(svc *orbit.EnrollmentService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpjson.Decode[orbit.EnrollRequest](r)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		host, nodeKey, err := svc.Enroll(r.Context(), req)
		switch {
		case errors.Is(err, orbit.ErrMissingHardwareUUID):
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, agentauth.ErrInvalidSecret):
			logger.WarnContext(
				r.Context(),
				"orbit enroll rejected", "operation", "enroll",
				"reason", "invalid_enroll_secret",
				"hardware_uuid", req.HardwareUUID,
			)
			httpjson.WriteError(w, http.StatusUnauthorized, "invalid enroll secret")
			return
		case err != nil:
			logger.ErrorContext(
				r.Context(),
				"orbit enroll failed", "operation", "enroll",
				"hardware_uuid", req.HardwareUUID,
				"err", err,
			)
			httpjson.WriteError(w, http.StatusInternalServerError, "enrollment failed")
			return
		}

		logger.InfoContext(
			r.Context(),
			"orbit host enrolled", "operation", "enroll",
			"host_id", host.ID,
			"hardware_uuid", host.Hardware.UUID,
			"display_name", host.DisplayName,
		)
		httpjson.Write(w, http.StatusOK, orbit.EnrollResponse{OrbitNodeKey: nodeKey})
	}
}

func orbitConfigHandler(svc *orbit.EnrollmentService, logger *slog.Logger) http.HandlerFunc {
	return orbitNodeKeyHandler(svc, logger,
		func(req orbit.ConfigRequest) string { return req.OrbitNodeKey },
		func(w http.ResponseWriter, r *http.Request, req orbit.ConfigRequest) {
			resp, err := svc.Config(r.Context(), req.OrbitNodeKey)
			if err != nil {
				logger.DebugContext(
					r.Context(),
					"orbit config rejected", "operation", "config",
					"reason", "invalid_node_key",
				)
				httpjson.WriteError(w, http.StatusUnauthorized, "invalid orbit node key")
				return
			}
			httpjson.Write(w, http.StatusOK, resp)
		})
}

func orbitDeviceMappingHandler(svc *orbit.EnrollmentService, logger *slog.Logger) http.HandlerFunc {
	return orbitNodeKeyHandler(svc, logger,
		func(req orbit.DeviceMappingRequest) string { return req.OrbitNodeKey },
		func(w http.ResponseWriter, r *http.Request, req orbit.DeviceMappingRequest) {
			if err := svc.SetUserAffinity(r.Context(), req.OrbitNodeKey, req.Email); err != nil {
				logger.WarnContext(
					r.Context(),
					"orbit device mapping rejected", "operation", "device_mapping",
					"reason", "invalid_node_key",
				)
				httpjson.WriteError(w, http.StatusUnauthorized, "invalid orbit node key")
				return
			}
			httpjson.Write(w, http.StatusOK, struct{}{})
		})
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

func registerOrbitCompatibilityRoutes(r chi.Router, svc *orbit.EnrollmentService, logger *slog.Logger) {
	r.Post(
		"/api/fleet/orbit/scripts/request",
		requireOrbitNodeKey(svc, logger, func(w http.ResponseWriter, _ *http.Request, _ orbitNodeKeyRequest) {
			httpjson.Write(w, http.StatusOK, scriptsRequestResponse{Scripts: []string{}})
		}),
	)
	r.Post("/api/fleet/orbit/scripts/result", requireOrbitNodeKey(svc, logger, orbitNoContentHandler))
	r.Post(
		"/api/fleet/orbit/software_install/details",
		requireOrbitNodeKey(svc, logger, orbitEmptyObjectHandler),
	)
	r.Post(
		"/api/fleet/orbit/software_install/package",
		requireOrbitNodeKey(svc, logger, orbitEmptyObjectHandler),
	)
	r.Post(
		"/api/fleet/orbit/software_install/result",
		requireOrbitNodeKey(svc, logger, orbitNoContentHandler),
	)
	r.Post(
		"/api/fleet/orbit/setup_experience/init",
		requireOrbitNodeKey(svc, logger, orbitEmptyObjectHandler),
	)
	r.Post(
		"/api/fleet/orbit/setup_experience/status",
		requireOrbitNodeKey(svc, logger, setupExperienceStatusHandler),
	)
	r.Post("/api/fleet/orbit/device_token", requireOrbitNodeKey(svc, logger, orbitNoContentHandler))
	r.Post(
		"/api/fleet/orbit/disk_encryption_key",
		requireOrbitNodeKey(svc, logger, orbitNoContentHandler),
	)
	r.Post("/api/fleet/orbit/luks_data", requireOrbitNodeKey(svc, logger, orbitNoContentHandler))
}

type orbitNodeKeyRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

func requireOrbitNodeKey(
	svc *orbit.EnrollmentService,
	logger *slog.Logger,
	next func(http.ResponseWriter, *http.Request, orbitNodeKeyRequest),
) http.HandlerFunc {
	return orbitNodeKeyHandler(svc, logger,
		func(req orbitNodeKeyRequest) string { return req.OrbitNodeKey },
		next,
	)
}

func orbitNodeKeyHandler[T any](
	svc *orbit.EnrollmentService,
	_ *slog.Logger,
	nodeKey func(T) string,
	handle func(http.ResponseWriter, *http.Request, T),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpjson.Decode[T](r)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := svc.ValidateNodeKey(r.Context(), nodeKey(req)); err != nil {
			httpjson.WriteError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		}
		handle(w, r, req)
	}
}

type scriptsRequestResponse struct {
	Scripts []string `json:"scripts"`
}

type setupExperienceResponse struct {
	SetupExperienceResults *struct{} `json:"setup_experience_results"`
	OK                     bool      `json:"ok"`
}

func orbitEmptyObjectHandler(w http.ResponseWriter, _ *http.Request, _ orbitNodeKeyRequest) {
	httpjson.Write(w, http.StatusOK, struct{}{})
}

func setupExperienceStatusHandler(w http.ResponseWriter, _ *http.Request, _ orbitNodeKeyRequest) {
	httpjson.Write(w, http.StatusOK, setupExperienceResponse{OK: true})
}

func orbitNoContentHandler(w http.ResponseWriter, _ *http.Request, _ orbitNodeKeyRequest) {
	w.WriteHeader(http.StatusNoContent)
}
