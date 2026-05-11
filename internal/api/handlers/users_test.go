package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/api/adminctx"
	"github.com/woodleighschool/woodstar/internal/users"
)

func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		wantStatus int
		wantOK     bool
	}{
		{
			name: "admin in context",
			ctx: adminctx.WithUser(
				context.Background(),
				&users.User{ID: 1, Role: users.RoleAdmin},
			),
			wantStatus: 0,
			wantOK:     true,
		},
		{
			name: "viewer is forbidden",
			ctx: adminctx.WithUser(
				context.Background(),
				&users.User{ID: 2, Role: users.RoleViewer},
			),
			wantStatus: 403,
		},
		{
			name:       "missing user is unauthorized",
			ctx:        context.Background(),
			wantStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := requireAdmin(tt.ctx)
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
			var status huma.StatusError
			if !errors.As(err, &status) {
				t.Fatalf("error is not huma.StatusError: %v", err)
			}
			if status.GetStatus() != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status.GetStatus(), tt.wantStatus)
			}
		})
	}
}

func TestParseUserID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{name: "positive", input: "42", want: 42},
		{name: "zero rejected", input: "0", wantErr: true},
		{name: "negative rejected", input: "-1", wantErr: true},
		{name: "non-numeric rejected", input: "abc", wantErr: true},
		{name: "empty rejected", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseUserID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestUserMutationErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not found", err: dbutil.ErrNotFound, wantStatus: 404},
		{name: "already exists", err: dbutil.ErrAlreadyExists, wantStatus: 409},
		{name: "weak password", err: users.ErrWeakPassword, wantStatus: 400},
		{name: "self role", err: users.ErrCannotChangeOwnRole, wantStatus: 409},
		{name: "self delete", err: users.ErrCannotDeleteSelf, wantStatus: 409},
		{name: "last admin", err: users.ErrCannotRemoveLastAdmin, wantStatus: 409},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mapped := userMutationError(tt.err)
			var status huma.StatusError
			if !errors.As(mapped, &status) {
				t.Fatalf("not a huma.StatusError: %v", mapped)
			}
			if status.GetStatus() != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status.GetStatus(), tt.wantStatus)
			}
		})
	}
}
