package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
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
			name:   "admin in context",
			ctx:    withUser(context.Background(), &users.User{ID: 1, Role: users.RoleAdmin}),
			wantOK: true,
		},
		{
			name:       "viewer is forbidden",
			ctx:        withUser(context.Background(), &users.User{ID: 2, Role: users.RoleViewer}),
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

func TestLoginInvalidCredentialsMessage(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := users.NewService(users.NewStore(database))
	if _, err := userService.Create(ctx, users.UserCreate{
		Email:    "admin@example.test",
		Name:     "Test Admin",
		Password: "correct-password",
		Role:     users.RoleAdmin,
	}); err != nil {
		t.Fatalf("create test user: %v", err)
	}

	router := chi.NewRouter()
	api := humachi.New(router, testHumaConfig())
	RegisterPublicAuth(api, auth.NewService(userService, testSessionManager()))

	body := strings.NewReader(`{"email":"admin@example.test","password":"wrong-password"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("invalid email or password")) {
		t.Fatalf("body = %q, want invalid credential message", rec.Body.String())
	}
}

func testSessionManager() *scs.SessionManager {
	sm := scs.New()
	sm.Store = memstore.New()
	return sm
}
