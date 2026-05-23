package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/users"
)

func TestSantaEventsAdminAPIListsAndFiltersEvents(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	store := santa.NewStore(db)
	service := santa.NewService(store)
	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-events-api-host",
		OrbitNodeKey: "santa-events-api-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	occurredAt := time.Date(2026, 5, 23, 14, 0, 0, 0, time.UTC)
	if _, err := service.HandleEventUpload(ctx, "santa-events-api-host", &syncv1.EventUploadRequest{
		MachineId: "santa-events-api-host",
		Events: []*syncv1.Event{
			{
				FileSha256:    "api-blocked",
				FileName:      "Blocked App",
				ExecutionTime: float64(occurredAt.Unix()),
				Decision:      syncv1.Decision_BLOCK_BINARY,
			},
			{
				FileSha256:    "api-allowed",
				FileName:      "Allowed App",
				ExecutionTime: float64(occurredAt.Add(time.Minute).Unix()),
				Decision:      syncv1.Decision_ALLOW_BINARY,
			},
			{
				FileSha256:    "api-blocked-certificate",
				FileName:      "Blocked Cert App",
				ExecutionTime: float64(occurredAt.Add(2 * time.Minute).Unix()),
				Decision:      syncv1.Decision_BLOCK_CERTIFICATE,
			},
		},
	}); err != nil {
		t.Fatalf("event upload: %v", err)
	}

	router, cookie := santaEventsRouter(t, db)
	rec := authedJSONRequest(
		t,
		router,
		cookie,
		http.MethodGet,
		"/api/santa/events?decision=blocked&limit=1",
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("events status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Blocked") ||
		strings.Contains(rec.Body.String(), "Allowed App") ||
		!strings.Contains(rec.Body.String(), "next_cursor") {
		t.Fatalf("events response = %q, want blocked page with cursor", rec.Body.String())
	}

	hostRec := authedJSONRequest(
		t,
		router,
		cookie,
		http.MethodGet,
		"/api/santa/events?host_id="+strconv.FormatInt(host.ID, 10)+"&decision=allow_binary",
		"",
	)
	if hostRec.Code != http.StatusOK {
		t.Fatalf("host events status = %d, want %d; body = %q", hostRec.Code, http.StatusOK, hostRec.Body.String())
	}
	if !strings.Contains(hostRec.Body.String(), "Allowed App") || strings.Contains(hostRec.Body.String(), "Blocked App") {
		t.Fatalf("host events response = %q, want exact decision filter", hostRec.Body.String())
	}
}

func santaEventsRouter(t *testing.T, db *database.DB) (*chi.Mux, *http.Cookie) {
	t.Helper()

	userService := users.NewService(users.NewStore(db))
	if _, err := userService.Create(context.Background(), users.CreateParams{
		Email:    "santa-events@example.test",
		Name:     "Santa Events Admin",
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
	RegisterSantaEvents(protected, santa.NewStore(db))

	return router, loginSantaEventsUser(t, authService, sessionManager)
}

func loginSantaEventsUser(t *testing.T, authService *auth.Service, sessionManager *scs.SessionManager) *http.Cookie {
	t.Helper()

	ctx, err := sessionManager.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, err := authService.Login(ctx, "santa-events@example.test", "correct-password"); err != nil {
		t.Fatalf("login test user: %v", err)
	}
	token, _, err := sessionManager.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return &http.Cookie{Name: sessionManager.Cookie.Name, Value: token}
}
