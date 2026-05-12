package protocol

import (
	"encoding/json"
	"net/http"
)

// writeJSON encodes body as JSON and writes it with the given status.
func writeJSON(w http.ResponseWriter, status int, body any) {
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

// writeError writes a {"error": message} body with the given status.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorBody{Error: message})
}

type errorBody struct {
	Error string `json:"error"`
}
