package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

func TestSantaConfigurationsAdminAPIManagesFullEditableShape(t *testing.T) {
	db, _ := dbtest.Open(t)
	labelID := createSantaConfigurationsAPILabel(t, db, "Santa Configurations API")
	router, cookie := santaConfigurationsRouter(t, db)

	createBody := fmt.Sprintf(`{
		"name": "API Config",
		"client_mode": "lockdown",
		"enable_bundles": true,
		"full_sync_interval_seconds": 120,
		"removable_media_action": "remount",
		"removable_media_remount_flags": ["rw", "nosuid"],
		"label_ids": [%d]
	}`, labelID)
	createRec := authedJSONRequest(t, router, cookie, http.MethodPost, "/api/santa/configurations", createBody)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %q", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var created santa.Configuration
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created configuration: %v", err)
	}
	if created.ID <= 0 || created.ClientMode != santa.ClientModeLockdown || len(created.LabelIDs) != 1 {
		t.Fatalf("created configuration = %+v", created)
	}

	conflictBody := fmt.Sprintf(`{"name": "Conflicting", "label_ids": [%d]}`, labelID)
	conflictRec := authedJSONRequest(t, router, cookie, http.MethodPost, "/api/santa/configurations", conflictBody)
	if conflictRec.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d, want %d; body = %q", conflictRec.Code, http.StatusConflict, conflictRec.Body.String())
	}
	if !strings.Contains(conflictRec.Body.String(), `"code":"configuration_label_conflict"`) ||
		!strings.Contains(conflictRec.Body.String(), `"configuration_name":"API Config"`) {
		t.Fatalf("conflict response = %q, want structured label conflict", conflictRec.Body.String())
	}

	listRec := authedJSONRequest(t, router, cookie, http.MethodGet, "/api/santa/configurations?q=API+Config", "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body = %q", listRec.Code, http.StatusOK, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), "API Config") {
		t.Fatalf("list response = %q, want created configuration", listRec.Body.String())
	}

	updateBody := fmt.Sprintf(`{
		"name": "Updated API Config",
		"client_mode": "monitor",
		"event_detail_url": "https://example.test/events",
		"label_ids": [%d]
	}`, labelID)
	updateRec := authedJSONRequest(
		t,
		router,
		cookie,
		http.MethodPatch,
		fmt.Sprintf("/api/santa/configurations/%d", created.ID),
		updateBody,
	)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body = %q", updateRec.Code, http.StatusOK, updateRec.Body.String())
	}
	var updated santa.Configuration
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated configuration: %v", err)
	}
	if updated.Name != "Updated API Config" || updated.EnableBundles != nil || updated.EventDetailURL == nil {
		t.Fatalf("updated configuration = %+v", updated)
	}

	deleteRec := authedJSONRequest(t, router, cookie, http.MethodDelete, fmt.Sprintf("/api/santa/configurations/%d", created.ID), "")
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d; body = %q", deleteRec.Code, http.StatusNoContent, deleteRec.Body.String())
	}
}

func santaConfigurationsRouter(t *testing.T, db *database.DB) (*chi.Mux, *http.Cookie) {
	t.Helper()

	userService := users.NewService(users.NewStore(db))
	if _, err := userService.Create(context.Background(), users.CreateParams{
		Email:    "santa-configurations@example.test",
		Name:     "Santa Configurations Admin",
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
	RegisterSantaConfigurations(protected, santa.NewStore(db))

	return router, loginSantaConfigurationsUser(t, authService, sessionManager)
}

func loginSantaConfigurationsUser(t *testing.T, authService *auth.Service, sessionManager *scs.SessionManager) *http.Cookie {
	t.Helper()

	ctx, err := sessionManager.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, err := authService.Login(ctx, "santa-configurations@example.test", "correct-password"); err != nil {
		t.Fatalf("login test user: %v", err)
	}
	token, _, err := sessionManager.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return &http.Cookie{Name: sessionManager.Cookie.Name, Value: token}
}

func createSantaConfigurationsAPILabel(t *testing.T, db *database.DB, name string) int64 {
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
