package ctxkeys

import (
	"context"
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/directory"
)

func TestRequireAdmin(t *testing.T) {
	admin := directory.RoleAdmin
	viewer := directory.RoleViewer
	tests := []struct {
		name       string
		user       *directory.User
		wantStatus int
		wantOK     bool
	}{
		{
			name:   "administrator in context",
			user:   &directory.User{ID: 1, Role: &admin},
			wantOK: true,
		},
		{
			name:       "viewer is forbidden",
			user:       &directory.User{ID: 2, Role: &viewer},
			wantStatus: 403,
		},
		{
			name:       "user without role is forbidden",
			user:       &directory.User{ID: 3},
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
				if got == nil || got.ID != tt.user.ID {
					t.Fatalf("user = %+v, want %+v", got, tt.user)
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

func TestCurrentUserID(t *testing.T) {
	if got := CurrentUserID(context.Background()); got != nil {
		t.Fatalf("anonymous user ID = %v, want nil", got)
	}
	ctx := WithUser(context.Background(), &directory.User{ID: 42})
	got := CurrentUserID(ctx)
	if got == nil || *got != 42 {
		t.Fatalf("current user ID = %v, want 42", got)
	}
}
