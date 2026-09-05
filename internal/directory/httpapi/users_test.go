package httpapi

import (
	"errors"
	"github.com/woodleighschool/woodstar/internal/directory"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/fault"
)

func TestUserMutationErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not found", err: fault.ErrNotFound, wantStatus: 404},
		{name: "already exists", err: fault.ErrAlreadyExists, wantStatus: 409},
		{name: "weak password", err: directory.ErrWeakPassword, wantStatus: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mapped := userMutationError(tt.err)
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
