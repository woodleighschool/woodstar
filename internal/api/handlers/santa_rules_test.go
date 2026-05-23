package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/platforms"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/users"
)

func TestSantaRulesAdminAPIManagesFullEditableShape(t *testing.T) {
	db, _ := dbtest.Open(t)
	labelID := createSantaRulesAPILabel(t, db, "Santa Rules API")
	router, cookie := santaRulesRouter(t, db)

	createBody := fmt.Sprintf(`{
		"rule_type": "binary",
		"identifier": "binary-sha",
		"name": "Example",
		"custom_message": "Blocked",
		"includes": [
			{"policy": "allowlist", "label_ids": [%d]}
		]
	}`, labelID)
	createRec := authedJSONRequest(t, router, cookie, http.MethodPost, "/api/santa/rules", createBody)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %q", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var created santa.Rule
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created rule: %v", err)
	}
	if created.ID <= 0 || len(created.Includes) != 1 {
		t.Fatalf("created rule = %+v", created)
	}

	listRec := authedJSONRequest(t, router, cookie, http.MethodGet, "/api/santa/rules?q=binary-sha", "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body = %q", listRec.Code, http.StatusOK, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), "binary-sha") {
		t.Fatalf("list response = %q, want created identifier", listRec.Body.String())
	}

	updateBody := fmt.Sprintf(`{
		"name": "Updated",
		"custom_url": "https://example.test",
		"includes": [
			{"policy": "blocklist", "label_ids": [%d]}
		],
		"exclude_label_ids": [%d]
	}`, labelID, labelID)
	updateRec := authedJSONRequest(
		t,
		router,
		cookie,
		http.MethodPatch,
		fmt.Sprintf("/api/santa/rules/%d", created.ID),
		updateBody,
	)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body = %q", updateRec.Code, http.StatusOK, updateRec.Body.String())
	}
	var updated santa.Rule
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated rule: %v", err)
	}
	if updated.RuleType != santa.RuleTypeBinary || updated.Identifier != "binary-sha" {
		t.Fatalf("update changed identity: %+v", updated)
	}
	if updated.Includes[0].Policy != santa.PolicyBlocklist || len(updated.ExcludeLabelIDs) != 1 {
		t.Fatalf("updated rule = %+v", updated)
	}

	deleteRec := authedJSONRequest(t, router, cookie, http.MethodDelete, fmt.Sprintf("/api/santa/rules/%d", created.ID), "")
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d; body = %q", deleteRec.Code, http.StatusNoContent, deleteRec.Body.String())
	}
}

func santaRulesRouter(t *testing.T, db *database.DB) (*chi.Mux, *http.Cookie) {
	t.Helper()

	userService := users.NewService(users.NewStore(db))
	if _, err := userService.Create(context.Background(), users.CreateParams{
		Email:    "santa-rules@example.test",
		Name:     "Santa Rules Admin",
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
	RegisterSantaRules(protected, santa.NewStore(db))

	return router, loginSantaRulesUser(t, authService, sessionManager)
}

func authedJSONRequest(
	t *testing.T,
	router *chi.Mux,
	cookie *http.Cookie,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewBufferString(body))
	req.AddCookie(cookie)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(rec, req)
	return rec
}

func loginSantaRulesUser(t *testing.T, authService *auth.Service, sessionManager *scs.SessionManager) *http.Cookie {
	t.Helper()

	ctx, err := sessionManager.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, err := authService.Login(ctx, "santa-rules@example.test", "correct-password"); err != nil {
		t.Fatalf("login test user: %v", err)
	}
	token, _, err := sessionManager.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return &http.Cookie{Name: sessionManager.Cookie.Name, Value: token}
}

func createSantaRulesAPILabel(t *testing.T, db *database.DB, name string) int64 {
	t.Helper()

	label, err := labels.NewStore(db).Create(t.Context(), labels.LabelCreate{
		Name:                name,
		LabelType:           labels.LabelTypeRegular,
		LabelMembershipType: labels.LabelMembershipTypeManual,
		Platforms: []platforms.Platform{
			platforms.PlatformDarwin,
			platforms.PlatformWindows,
			platforms.PlatformLinux,
		},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	return label.ID
}
