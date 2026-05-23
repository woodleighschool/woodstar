package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/platforms"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/users"
)

func TestHostDetailOmitsSantaUntilObserved(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	santaStore := santa.NewStore(db)
	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "host-detail-no-santa",
		OrbitNodeKey: "host-detail-no-santa-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	router, cookie := santaHostDetailRouter(t, db, hostStore, santaStore)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("/api/hosts/%d", host.ID),
		nil,
	)
	req.AddCookie(cookie)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host detail: %v", err)
	}
	if _, ok := body["santa"]; ok {
		t.Fatalf("host detail included santa before observation: %q", rec.Body.String())
	}
	if got := int64(body["id"].(float64)); got != host.ID {
		t.Fatalf("host id = %d, want %d", got, host.ID)
	}
}

func TestHostDetailIncludesSantaObservation(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	santaStore := santa.NewStore(db)
	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID:   "host-detail-with-santa",
		HardwareSerial: "SANTADETAIL",
		OrbitNodeKey:   "host-detail-with-santa-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	seenAt := time.Date(2026, 5, 23, 11, 0, 0, 0, time.UTC)
	if err := santaStore.UpsertHostObservation(ctx, santa.HostObservation{
		HostID:             host.ID,
		MachineID:          "host-detail-with-santa",
		SerialNumber:       "SANTADETAIL",
		Version:            "2026.2",
		ClientModeReported: santa.ClientModeLockdown,
		LastSeenAt:         &seenAt,
	}); err != nil {
		t.Fatalf("upsert santa observation: %v", err)
	}
	label, err := labels.NewStore(db).Create(ctx, labels.LabelCreate{
		Name:                "Host Detail Santa Configuration",
		LabelType:           labels.LabelTypeRegular,
		LabelMembershipType: labels.LabelMembershipTypeManual,
		Platforms: []platforms.Platform{
			platforms.PlatformDarwin,
			platforms.PlatformWindows,
			platforms.PlatformLinux,
		},
	})
	if err != nil {
		t.Fatalf("create configuration label: %v", err)
	}
	if err := labels.NewStore(db).SetMembership(ctx, label.ID, host.ID, true); err != nil {
		t.Fatalf("set configuration label membership: %v", err)
	}
	configuration, err := santaStore.CreateConfiguration(ctx, santa.ConfigurationCreate{
		Name:     "Host Detail Config",
		LabelIDs: []int64{label.ID},
	})
	if err != nil {
		t.Fatalf("create santa configuration: %v", err)
	}

	router, cookie := santaHostDetailRouter(t, db, hostStore, santaStore)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("/api/hosts/%d", host.ID),
		nil,
	)
	req.AddCookie(cookie)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body struct {
		Santa *struct {
			Enrolled               bool   `json:"enrolled"`
			Version                string `json:"version"`
			ClientModeReported     string `json:"client_mode_reported"`
			EffectiveConfiguration *struct {
				ID              int64 `json:"id"`
				MatchedViaLabel *struct {
					ID   int64  `json:"id"`
					Name string `json:"name"`
				} `json:"matched_via_label"`
			} `json:"effective_configuration"`
		} `json:"santa"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host detail: %v", err)
	}
	if body.Santa == nil {
		t.Fatalf("host detail did not include santa: %q", rec.Body.String())
	}
	if !body.Santa.Enrolled || body.Santa.Version != "2026.2" || body.Santa.ClientModeReported != "lockdown" {
		t.Fatalf("santa detail = %+v", body.Santa)
	}
	if body.Santa.EffectiveConfiguration == nil || body.Santa.EffectiveConfiguration.ID != configuration.ID {
		t.Fatalf("effective configuration = %+v, want %d", body.Santa.EffectiveConfiguration, configuration.ID)
	}
	if body.Santa.EffectiveConfiguration.MatchedViaLabel == nil ||
		body.Santa.EffectiveConfiguration.MatchedViaLabel.ID != label.ID ||
		body.Santa.EffectiveConfiguration.MatchedViaLabel.Name != label.Name {
		t.Fatalf("matched label = %+v, want %s", body.Santa.EffectiveConfiguration.MatchedViaLabel, label.Name)
	}
}

func santaHostDetailRouter(
	t *testing.T,
	db *database.DB,
	hostStore *hosts.Store,
	santaStore *santa.Store,
) (*chi.Mux, *http.Cookie) {
	t.Helper()

	userService := users.NewService(users.NewStore(db))
	if _, err := userService.Create(context.Background(), users.CreateParams{
		Email:    "host-detail-admin@example.test",
		Name:     "Host Detail Admin",
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
	RegisterHosts(protected, hostStore, nil, santaStore)

	return router, loginHostDetailTestUser(t, authService, sessionManager)
}

func loginHostDetailTestUser(t *testing.T, authService *auth.Service, sessionManager *scs.SessionManager) *http.Cookie {
	t.Helper()

	ctx, err := sessionManager.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, err := authService.Login(ctx, "host-detail-admin@example.test", "correct-password"); err != nil {
		t.Fatalf("login test user: %v", err)
	}
	token, _, err := sessionManager.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return &http.Cookie{Name: sessionManager.Cookie.Name, Value: token}
}
