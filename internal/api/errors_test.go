package api

import (
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

func TestHumaErrorLogsInternalErrors(t *testing.T) {
	var logBuf strings.Builder
	installHumaErrorHandler(slog.New(slog.NewTextHandler(&logBuf, nil)))

	raw := errors.New("get session user: column \"api_key\" does not exist")
	got := huma.NewErrorWithContext(nil, 500, "unexpected error", raw)

	model, ok := got.(*huma.ErrorModel)
	if !ok {
		t.Fatalf("got %T, want *huma.ErrorModel", got)
	}
	if model.Status != 500 {
		t.Fatalf("status = %d, want 500", model.Status)
	}
	if len(model.Errors) != 1 || !strings.Contains(model.Errors[0].Message, "does not exist") {
		t.Fatalf("expected raw error to pass through to body, got %+v", model.Errors)
	}
	logs := logBuf.String()
	if !strings.Contains(logs, "get session user") || !strings.Contains(logs, "does not exist") {
		t.Fatalf("expected raw error in logs, got %q", logs)
	}
}

func TestHumaErrorDoesNotLog4xx(t *testing.T) {
	var logBuf strings.Builder
	installHumaErrorHandler(slog.New(slog.NewTextHandler(&logBuf, nil)))

	huma.NewErrorWithContext(nil, 400, "validation failed", errors.New("name is required"))

	if logBuf.Len() != 0 {
		t.Fatalf("expected no log output for 4xx, got %q", logBuf.String())
	}
}
