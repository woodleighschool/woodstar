package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

// installHumaErrorHandler overrides huma's NewErrorWithContext so that 5xx
// responses also produce an ERROR-level log entry on the server. The body
// content is unchanged: admins still see the underlying error text in the
// HTTP response, matching the "log and surface" pattern Fleet uses for
// admin-only APIs.
func installHumaErrorHandler(logger *slog.Logger) {
	huma.NewErrorWithContext = func(hctx huma.Context, status int, msg string, errs ...error) huma.StatusError {
		if status >= http.StatusInternalServerError {
			ctx := context.Background()
			if hctx != nil {
				ctx = hctx.Context()
			}
			attrs := []any{"status", status, "msg", msg}
			for i, e := range errs {
				if e == nil {
					continue
				}
				attrs = append(attrs, "err"+strconv.Itoa(i), e.Error())
			}
			logger.ErrorContext(ctx, "handler error", attrs...)
		}
		return huma.NewError(status, msg, errs...)
	}
}
