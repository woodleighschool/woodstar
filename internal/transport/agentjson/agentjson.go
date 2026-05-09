// Package agentjson writes JSON responses for the Orbit and osquery agent
// endpoints. These endpoints don't go through Huma so they need a small
// shared writer.
package agentjson

import (
	"encoding/json"
	"net/http"
)

// Write encodes body as JSON and writes it with the given status.
func Write(w http.ResponseWriter, status int, body any) {
	payload, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(append(payload, '\n')); err != nil {
		return
	}
}

// WriteError writes a {"error": message} body with the given status.
func WriteError(w http.ResponseWriter, status int, message string) {
	Write(w, status, errorBody{Error: message})
}

type errorBody struct {
	Error string `json:"error"`
}
