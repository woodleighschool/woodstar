package protocol

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/httpjson"
	"github.com/woodleighschool/woodstar/internal/orbit"
)

func registerOrbitCompatibilityRoutes(r chi.Router, svc *orbit.EnrollmentService) {
	r.Post(
		"/api/fleet/orbit/scripts/request",
		requireOrbitNodeKey(svc, func(w http.ResponseWriter, _ *http.Request, _ orbitNodeKeyRequest) {
			httpjson.Write(w, http.StatusOK, scriptsRequestResponse{Scripts: []string{}})
		}),
	)
	r.Post("/api/fleet/orbit/scripts/result", requireOrbitNodeKey(svc, orbitNoContentHandler))
	r.Post(
		"/api/fleet/orbit/software_install/details",
		requireOrbitNodeKey(svc, orbitEmptyObjectHandler),
	)
	r.Post(
		"/api/fleet/orbit/software_install/package",
		requireOrbitNodeKey(svc, orbitEmptyObjectHandler),
	)
	r.Post(
		"/api/fleet/orbit/software_install/result",
		requireOrbitNodeKey(svc, orbitNoContentHandler),
	)
	r.Post(
		"/api/fleet/orbit/setup_experience/init",
		requireOrbitNodeKey(svc, orbitEmptyObjectHandler),
	)
	r.Post(
		"/api/fleet/orbit/setup_experience/status",
		requireOrbitNodeKey(svc, setupExperienceStatusHandler),
	)
	r.Post("/api/fleet/orbit/device_token", requireOrbitNodeKey(svc, orbitNoContentHandler))
	r.Post(
		"/api/fleet/orbit/disk_encryption_key",
		requireOrbitNodeKey(svc, orbitNoContentHandler),
	)
	r.Post("/api/fleet/orbit/luks_data", requireOrbitNodeKey(svc, orbitNoContentHandler))
}

type orbitNodeKeyRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

func requireOrbitNodeKey(
	svc *orbit.EnrollmentService,
	next func(http.ResponseWriter, *http.Request, orbitNodeKeyRequest),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httpjson.Decode[orbitNodeKeyRequest](r)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := svc.ValidateNodeKey(r.Context(), req.OrbitNodeKey); err != nil {
			httpjson.WriteError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		}
		next(w, r, req)
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
