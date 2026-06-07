package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/adminapi/adminctx"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/references"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestSoftwareSantaReferenceEndpoint(t *testing.T) {
	db, ctx := dbtest.Open(t)
	router, protected, cookie := santaTestAPIWith(t, db, "software-santa-admin@example.test", false)
	RegisterSoftware(protected, inventory.NewStore(db), references.NewStore(db))

	var titleID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO software_titles (name, display_name, source, bundle_identifier)
		VALUES ('Reference Endpoint', 'Reference Endpoint', 'apps', 'com.example.reference-endpoint')
		RETURNING id
	`).Scan(&titleID); err != nil {
		t.Fatalf("insert software title: %v", err)
	}

	rec := santaAdminRequest(
		t,
		router,
		cookie,
		http.MethodGet,
		fmt.Sprintf("/api/software/%d/santa", titleID),
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body references.SoftwareReference
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode software reference: %v", err)
	}
	if body.ExecutionCount != 0 || body.BlockCount != 0 {
		t.Fatalf("software reference = %+v, want empty counts", body)
	}
}

func TestHostDetailRunsSantaEnricher(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	santaStore := santa.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware: hosts.HostHardware{
			UUID:   "santa-enricher-host",
			Serial: "ENRICH",
		},
		OrbitNodeKey: "santa-enricher-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	seenAt := time.Date(2026, 5, 23, 11, 0, 0, 0, time.UTC)
	if err := santaStore.UpsertHostObservation(ctx, santa.HostObservation{
		HostID:             host.ID,
		MachineID:          "santa-enricher-host",
		SerialNumber:       "ENRICH",
		Version:            "2026.2",
		ClientModeReported: configurations.ReportedClientModeLockdown,
		LastSeenAt:         &seenAt,
	}); err != nil {
		t.Fatalf("upsert santa observation: %v", err)
	}
	label, err := labels.NewStore(db).Create(ctx, labels.LabelMutation{
		Name:                "Enricher Label",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	if err := labels.NewStore(db).SetMembership(ctx, label.ID, host.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}
	configuration, err := configurations.NewStore(db).CreateConfiguration(ctx, configurations.ConfigurationMutation{
		Name:                    "Enricher Config",
		ClientMode:              configurations.ClientModeMonitor,
		FullSyncIntervalSeconds: 600,
		BatchSize:               50,
		Targets: configurations.ConfigurationTargets{
			Include: []targeting.LabelRef{{LabelID: label.ID}},
		},
	})
	if err != nil {
		t.Fatalf("create configuration: %v", err)
	}

	hostState := santa.NewHostStateService(santaStore, configurations.NewStore(db))
	router, cookie := santaAuthedRouter(t, db, "enricher-admin@example.test", func(api huma.API) {
		RegisterHosts(
			api,
			hostStore,
			hosts.NewUserAffinityStore(db),
			nil,
			checks.NewStore(db),
			SantaHostDetailContributor(hostState),
		)
	})
	rec := santaAdminRequest(t, router, cookie, http.MethodGet, fmt.Sprintf("/api/hosts/%d", host.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Santa *struct {
			Version            string `json:"version"`
			ClientModeReported string `json:"client_mode_reported"`
			Configuration      *struct {
				ID              int64 `json:"id"`
				MatchedViaLabel *struct {
					ID int64 `json:"id"`
				} `json:"matched_via_label"`
				Targets configurations.ConfigurationTargets `json:"targets"`
			} `json:"configuration"`
		} `json:"santa"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host detail: %v", err)
	}
	if body.Santa == nil {
		t.Fatalf("enricher did not fire; body = %q", rec.Body.String())
	}
	if body.Santa.Version != "2026.2" || body.Santa.ClientModeReported != "lockdown" {
		t.Fatalf("santa observation = %+v", body.Santa)
	}
	if body.Santa.Configuration == nil ||
		body.Santa.Configuration.ID != configuration.ID ||
		body.Santa.Configuration.MatchedViaLabel == nil ||
		body.Santa.Configuration.MatchedViaLabel.ID != label.ID {
		t.Fatalf("configuration = %+v, want id=%d via label=%d",
			body.Santa.Configuration, configuration.ID, label.ID)
	}
	if len(body.Santa.Configuration.Targets.Include) != 1 ||
		body.Santa.Configuration.Targets.Include[0].LabelID != label.ID ||
		len(body.Santa.Configuration.Targets.Exclude) != 0 {
		t.Fatalf("configuration targets = %+v, want canonical target set", body.Santa.Configuration.Targets)
	}
}

func santaAuthedRouter(t *testing.T, db *database.DB, email string, register func(huma.API)) (*chi.Mux, *http.Cookie) {
	t.Helper()
	router, protected, cookie := santaTestAPIWith(t, db, email, false)
	register(protected)
	return router, cookie
}

func santaTestAPIWith(
	t *testing.T,
	db *database.DB,
	email string,
	requireAdminGroup bool,
) (*chi.Mux, huma.API, *http.Cookie) {
	t.Helper()

	userService := directory.NewUserService(directory.NewStore(db))
	admin, err := userService.Create(context.Background(), directory.UserCreate{
		Email:    email,
		Name:     "Santa Test Admin",
		Password: "correct-password",
		Role:     directory.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	sessionManager := testSessionManager()
	authService := auth.NewService(userService, sessionManager)
	router := chi.NewRouter()
	router.Use(sessionManager.LoadAndSave)
	api := humachi.New(router, testHumaConfig())
	protected := huma.NewGroup(api)
	protected.UseMiddleware(santaTestWithUser(admin))
	if requireAdminGroup {
		protected.UseMiddleware(santaTestRequireAdmin)
	}

	return router, protected, loginSantaTestUser(t, authService, sessionManager, email)
}

func santaTestWithUser(user *directory.User) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, adminctx.WithUser(ctx.Context(), user)))
	}
}

func santaTestRequireAdmin(ctx huma.Context, next func(huma.Context)) {
	if _, err := adminctx.RequireAdmin(ctx.Context()); err != nil {
		panic(err)
	}
	next(ctx)
}

func loginSantaTestUser(
	t *testing.T,
	authService *auth.Service,
	sessionManager *scs.SessionManager,
	email string,
) *http.Cookie {
	t.Helper()

	ctx, err := sessionManager.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, err := authService.Login(ctx, email, "correct-password"); err != nil {
		t.Fatalf("login test user: %v", err)
	}
	token, _, err := sessionManager.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return &http.Cookie{Name: sessionManager.Cookie.Name, Value: token}
}

func testSessionManager() *scs.SessionManager {
	sm := scs.New()
	sm.Store = memstore.New()
	return sm
}

func santaAdminRequest(
	t *testing.T,
	router *chi.Mux,
	cookie *http.Cookie,
	method, path, body string,
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
