package protocol

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

var (
	errUnauthorized      = errors.New("unauthorized santa sync request")
	errUnsupportedMedia  = errors.New("unsupported santa sync media")
	errInvalidSyncBody   = errors.New("invalid santa sync request body")
	errRequestBodyTooBig = errors.New("santa sync request body too large")
)

func (h handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := statusCodeForError(err)
	h.log(r, statusCode, err)
	writeStatusOnly(w, statusCode)
}

func statusCodeForError(err error) int {
	switch {
	case errors.Is(err, errUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, errUnsupportedMedia):
		return http.StatusUnsupportedMediaType
	case errors.Is(err, errInvalidSyncBody),
		errors.Is(err, errRequestBodyTooBig),
		errors.Is(err, dbutil.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, dbutil.ErrNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeStatusOnly(w http.ResponseWriter, statusCode int) {
	w.Header().Del("Content-Type")
	w.Header().Del("Content-Encoding")
	w.WriteHeader(statusCode)
}

func (h handler) log(r *http.Request, statusCode int, err error) {
	args := []any{
		"status", statusCode,
		"method", r.Method,
		"path", r.URL.Path,
		"machine_id", chi.URLParam(r, "machine_id"),
		"err", err,
	}
	if statusCode >= http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "santa sync request failed", args...)
		return
	}
	h.logger.WarnContext(r.Context(), "santa sync request rejected", args...)
}
