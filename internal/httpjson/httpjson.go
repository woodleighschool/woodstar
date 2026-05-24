// Package httpjson provides small JSON helpers for raw net/http endpoints.
package httpjson

import (
	"encoding/json"
	"net/http"
)

type ErrorBody struct {
	Error string `json:"error"`
}

// Write encodes body as JSON and writes it with the given status. Write
// failures are dropped: when the client has hung up there is nothing useful
// the handler can do.
func Write(w http.ResponseWriter, status int, body any) {
	payload, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(payload, '\n'))
}

func WriteError(w http.ResponseWriter, status int, message string) {
	Write(w, status, ErrorBody{Error: message})
}

func Decode[T any](r *http.Request) (T, error) {
	var req T
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}
