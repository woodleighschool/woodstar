package ctxkeys

import (
	"context"
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/directory"
)

func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name       string
		principal  *auth.Principal
		wantStatus int
		wantOK     bool
	}{
		{
			name:      "initial administrator in context",
			principal: &auth.Principal{Role: directory.RoleAdmin},
			wantOK:    true,
		},
		{
			name:       "viewer is forbidden",
			principal:  &auth.Principal{Role: directory.RoleViewer},
			wantStatus: 403,
		},
		{
			name:       "missing principal is unauthorized",
			wantStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if tt.principal != nil {
				ctx = WithPrincipal(ctx, tt.principal)
			}
			got, err := RequireAdmin(ctx)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Fatal("expected principal, got nil")
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			status, ok := errors.AsType[huma.StatusError](err)
			if !ok {
				t.Fatalf("error is not huma.StatusError: %v", err)
			}
			if status.GetStatus() != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status.GetStatus(), tt.wantStatus)
			}
		})
	}
}

func TestRequireUserIDRejectsInitialAdmin(t *testing.T) {
	ctx := WithPrincipal(context.Background(), &auth.Principal{Role: directory.RoleAdmin})
	_, err := RequireUserID(ctx)
	status, ok := errors.AsType[huma.StatusError](err)
	if !ok || status.GetStatus() != 404 {
		t.Fatalf("RequireUserID error = %v, want 404", err)
	}
}
