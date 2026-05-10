package transport

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func requestLogger(logger *slog.Logger, accessLevel slog.Level) func(http.Handler) http.Handler {
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
						"path", r.URL.Path,
						"recover", recovered,
						"stack", string(debug.Stack()),
					)
					http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				logRequest(r.Context(), logger, r, ww, time.Since(started), accessLevel)
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
	accessLevel slog.Level,
) {
	status := ww.Status()
	if status == 0 {
		status = http.StatusOK
	}

	attrs := []any{
		"request_id", chimiddleware.GetReqID(ctx),
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"duration_ms", float64(elapsed.Microseconds()) / 1000,
		"remote_addr", r.RemoteAddr,
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
		logger.Log(ctx, accessLevel, "request completed", attrs...)
	}
}
