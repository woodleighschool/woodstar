package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/users"
)

func TestSantaSyncTokenAdminAPICreatesPlaintextOnce(t *testing.T) {
	db, ctx := dbtest.Open(t)
	userService := users.NewService(users.NewStore(db))
	if _, err := userService.Create(ctx, users.CreateParams{
		Email:    "admin@example.test",
		Name:     "Admin",
		Password: "correct-password",
		Role:     users.RoleAdmin,
	}); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	sessionManager := testSessionManager()
	authService := auth.NewService(userService, sessionManager)
	router := chi.NewRouter()
	router.Use(sessionManager.LoadAndSave)
	api := humachi.New(router, huma.DefaultConfig("test", "test"))
	protected := huma.NewGroup(api)
	protected.UseMiddleware(RequireAuth(api, authService))
	RegisterSantaSyncTokens(protected, santa.NewStore(db))

	cookie := loginSantaTokenTestUser(t, authService, sessionManager)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/santa/sync-tokens", nil)
	createReq.AddCookie(cookie)
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %q", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var created struct {
		ID        int64  `json:"id"`
		Value     string `json:"value"`
		ValueHash string `json:"value_hash"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID <= 0 || created.Value == "" || created.ValueHash == "" {
		t.Fatalf("create response = %+v, want id, value, and value_hash", created)
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/santa/sync-tokens", nil)
	listReq.AddCookie(cookie)
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body = %q", listRec.Code, http.StatusOK, listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), created.Value) {
		t.Fatalf("list response leaked plaintext token: %q", listRec.Body.String())
	}
	var listed []struct {
		ID        int64  `json:"id"`
		ValueHash string `json:"value_hash"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID || listed[0].ValueHash != created.ValueHash {
		t.Fatalf("list response = %+v, want created token metadata", listed)
	}
}

func loginSantaTokenTestUser(t *testing.T, authService *auth.Service, sessionManager *scs.SessionManager) *http.Cookie {
	t.Helper()

	ctx, err := sessionManager.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, err := authService.Login(ctx, "admin@example.test", "correct-password"); err != nil {
		t.Fatalf("login test user: %v", err)
	}
	token, _, err := sessionManager.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return &http.Cookie{Name: sessionManager.Cookie.Name, Value: token}
}
