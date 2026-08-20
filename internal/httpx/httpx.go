package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

type ErrorBody struct {
	Error string `json:"error"`
}

// BearerToken parses a single-token Bearer Authorization header.
func BearerToken(authorization string) (string, bool) {
	scheme, value, ok := strings.Cut(authorization, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		return "", false
	}
	return value, true
}

// PathParam returns a chi route parameter decoded to the same once-unescaped
// form as r.URL.Path. It returns an empty string when the parameter is invalid.
func PathParam(r *http.Request, key string) string {
	value := chi.URLParam(r, key)
	if r.URL.RawPath == "" {
		return value
	}
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return ""
	}
	return decoded
}

// EscapePath percent-encodes each segment of a logical slash-separated path.
func EscapePath(value string) string {
	parts := strings.Split(value, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
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

func Decode[T any](w http.ResponseWriter, r *http.Request, maxBytes int64) (T, error) {
	var req T
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	if err := decoder.Decode(&req); err != nil {
		return req, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("request body contains multiple JSON values")
		}
		return req, err
	}
	return req, nil
}

func WriteDecodeError(w http.ResponseWriter, err error) {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	WriteError(w, http.StatusBadRequest, "invalid request body")
}
