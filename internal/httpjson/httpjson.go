// Package httpjson provides small JSON helpers for raw net/http endpoints.
package httpjson

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

type ErrorBody struct {
	Error string `json:"error"`
}

func Write(w http.ResponseWriter, status int, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(append(payload, '\n'))
	return err
}

func WriteError(w http.ResponseWriter, status int, message string) error {
	return Write(w, status, ErrorBody{Error: message})
}

func Decode[T any](r *http.Request) (T, error) {
	var req T
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func LogWriteError(ctx context.Context, logger *slog.Logger, message string, err error) {
	if err == nil || logger == nil {
		return
	}
	logger.DebugContext(ctx, message, "err", err)
}
