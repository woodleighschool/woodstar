package orbit

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	coreorbit "github.com/woodleighschool/woodstar/internal/agents/orbit"
)

func registerStubs(r chi.Router, svc *coreorbit.Service) {
	r.Post("/api/fleet/orbit/scripts/request", requireNodeKey(svc, func(w http.ResponseWriter, _ *http.Request) {
		writeAgentJSON(w, http.StatusOK, scriptsRequestResponse{Scripts: []string{}})
	}))
	r.Post("/api/fleet/orbit/scripts/result", requireNodeKey(svc, noContentHandler))
	r.Post("/api/fleet/orbit/software_install/details", requireNodeKey(svc, emptyObjectHandler))
	r.Post("/api/fleet/orbit/software_install/package", requireNodeKey(svc, emptyObjectHandler))
	r.Post("/api/fleet/orbit/software_install/result", requireNodeKey(svc, noContentHandler))
	r.Post("/api/fleet/orbit/setup_experience/init", requireNodeKey(svc, emptyObjectHandler))
	r.Post("/api/fleet/orbit/setup_experience/status", requireNodeKey(svc, setupExperienceStatusHandler))
	r.Post("/api/fleet/orbit/device_token", requireNodeKey(svc, noContentHandler))
	r.Post("/api/fleet/orbit/disk_encryption_key", requireNodeKey(svc, noContentHandler))
	r.Post("/api/fleet/orbit/luks_data", requireNodeKey(svc, noContentHandler))
}

func requireNodeKey(svc *coreorbit.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OrbitNodeKey string `json:"orbit_node_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := svc.ValidateNodeKey(r.Context(), req.OrbitNodeKey); err != nil {
			writeAgentError(w, http.StatusUnauthorized, "invalid orbit node key")
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

func emptyObjectHandler(w http.ResponseWriter, _ *http.Request) {
	writeAgentJSON(w, http.StatusOK, struct{}{})
}

func setupExperienceStatusHandler(w http.ResponseWriter, _ *http.Request) {
	writeAgentJSON(w, http.StatusOK, setupExperienceResponse{OK: true})
}

func noContentHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(capabilitiesHeader, orbitCapabilities)
	w.WriteHeader(http.StatusNoContent)
}
