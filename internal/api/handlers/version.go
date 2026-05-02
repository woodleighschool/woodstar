package handlers

import (
	"net/http"
	"time"
)

type VersionHandler struct {
	version string
	started time.Time
}

func NewVersionHandler(version string, started time.Time) *VersionHandler {
	return &VersionHandler{
		version: version,
		started: started,
	}
}

func (h *VersionHandler) Current(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":    h.version,
		"started_at": h.started.Format(time.RFC3339),
	})
}
