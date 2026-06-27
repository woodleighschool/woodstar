package handlers

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestResourceMutationErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not found", err: dbutil.ErrNotFound, wantStatus: 404},
		{name: "already exists", err: dbutil.ErrAlreadyExists, wantStatus: 409},
		{name: "validation", err: dbutil.ErrInvalidInput, wantStatus: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mapped := ResourceMutationError("resource", tt.err)
			status, ok := errors.AsType[huma.StatusError](mapped)
			if !ok {
				t.Fatalf("not a huma.StatusError: %v", mapped)
			}
			if status.GetStatus() != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status.GetStatus(), tt.wantStatus)
			}
		})
	}
}

func TestHandlerErrorLogsInternalErrors(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	err := errors.New("database unavailable")

	got := handlerError(context.Background(), logger, "list-users", err, "user_id", int64(7))
	if !errors.Is(got, err) {
		t.Fatalf("handlerError returned %v, want original error", got)
	}

	line := buf.String()
	for _, want := range []string{
		`"msg":"api handler failed"`,
		`"operation":"list-users"`,
		`"status":500`,
		`"user_id":7`,
		`"err":"database unavailable"`,
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("log line %q does not contain %s", line, want)
		}
	}
}

func TestHandlerErrorDoesNotLogClientErrors(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	err := huma.Error404NotFound("missing")
	if got := handlerError(context.Background(), logger, "get-user", err); !errors.Is(got, err) {
		t.Fatalf("handlerError returned %v, want original error", got)
	}
	if buf.Len() != 0 {
		t.Fatalf("client error logged: %s", buf.String())
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
