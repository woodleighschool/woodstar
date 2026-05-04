package orbit

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func registerStubs(r chi.Router, svc *Service) {
	r.Post("/api/fleet/orbit/scripts/request", requireNodeKey(svc, func(w http.ResponseWriter, _ *http.Request) {
		writeAgentJSON(w, http.StatusOK, ScriptsRequestResponse{Scripts: []any{}})
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

func requireNodeKey(svc *Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OrbitNodeKey string `json:"orbit_node_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAgentError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if _, err := svc.hosts.GetByOrbitNodeKey(r.Context(), req.OrbitNodeKey); err != nil {
			writeAgentError(w, http.StatusUnauthorized, "invalid orbit node key")
			return
		}
		next(w, r)
	}
}

func emptyObjectHandler(w http.ResponseWriter, _ *http.Request) {
	writeAgentJSON(w, http.StatusOK, map[string]any{})
}

func setupExperienceStatusHandler(w http.ResponseWriter, _ *http.Request) {
	writeAgentJSON(w, http.StatusOK, map[string]any{
		"setup_experience_results": nil,
		"ok":                       true,
	})
}

func noContentHandler(w http.ResponseWriter, _ *http.Request) {
	writeOrbitHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}
