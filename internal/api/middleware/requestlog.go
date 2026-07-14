package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestLogger returns a middleware that logs every HTTP request.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	logger = logger.With("component", "http")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(
						r.Context(),
						"panic recovered",
						"method", r.Method,
						"path", requestLogPath(r),
						"recover", recovered,
						"stack", string(debug.Stack()),
					)
					http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				logRequest(r.Context(), logger, r, ww, time.Since(started))
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

func logRequest(
	ctx context.Context,
	logger *slog.Logger,
	r *http.Request,
	ww chimiddleware.WrapResponseWriter,
	elapsed time.Duration,
) {
	status := ww.Status()
	if status == 0 {
		status = http.StatusOK
	}

	attrs := []any{
		"request_id", chimiddleware.GetReqID(ctx),
		"method", r.Method,
		"path", requestLogPath(r),
		"status", status,
		"duration_ms", float64(elapsed.Microseconds()) / 1000,
		"client_ip", chimiddleware.GetClientIP(ctx),
	}
	if userAgent := r.UserAgent(); userAgent != "" {
		attrs = append(attrs, "user_agent", userAgent)
	}

	switch {
	case status >= 500:
		logger.ErrorContext(ctx, "request failed", attrs...)
	case status >= 400:
		logger.WarnContext(ctx, "request rejected", attrs...)
	default:
		logger.DebugContext(ctx, "request completed", attrs...)
	}
}

func requestLogPath(r *http.Request) string {
	if pattern := chi.RouteContext(r.Context()).RoutePattern(); pattern != "" {
		return pattern
	}

	const devicePrefix = "/api/latest/fleet/device/"
	if suffix, ok := strings.CutPrefix(r.URL.Path, devicePrefix); ok {
		if _, remainder, found := strings.Cut(suffix, "/"); found {
			return devicePrefix + "{token}/" + remainder
		}
		if suffix != "" {
			return devicePrefix + "{token}"
		}
	}
	return r.URL.Path
}
