package ctxkeys

import (
	"context"
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/directory"
)

func TestRequireAdmin(t *testing.T) {
	adminRole := directory.RoleAdmin
	viewerRole := directory.RoleViewer
	tests := []struct {
		name       string
		user       *directory.User
		wantStatus int
		wantOK     bool
	}{
		{
			name:   "admin in context",
			user:   &directory.User{ID: 1, Role: &adminRole},
			wantOK: true,
		},
		{
			name:       "viewer is forbidden",
			user:       &directory.User{ID: 2, Role: &viewerRole},
			wantStatus: 403,
		},
		{
			name:       "missing user is unauthorized",
			wantStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if tt.user != nil {
				ctx = WithUser(ctx, tt.user)
			}
			got, err := RequireAdmin(ctx)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Fatal("expected user, got nil")
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
