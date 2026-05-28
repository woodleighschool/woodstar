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
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	"github.com/woodleighschool/woodstar/internal/users"
)

func TestSantaConfigurationLabelConflictResponseShape(t *testing.T) {
	db, ctx := dbtest.Open(t)
	router, protected, cookie := santaAdminTestAPI(t, db, "config-conflict-admin@example.test")
	RegisterSantaConfigurations(protected, configurations.NewStore(db))

	label, err := labels.NewStore(db).Create(ctx, labels.LabelCreate{
		Name:                "Conflict Label",
		LabelType:           labels.LabelTypeRegular,
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}

	body := fmt.Sprintf(`{"name":"Owner","label_ids":[%d]}`, label.ID)
	if rec := santaAdminRequest(
		t,
		router,
		cookie,
		http.MethodPost,
		"/api/santa/configurations",
		body,
	); rec.Code != http.StatusCreated {
		t.Fatalf("seed configuration status = %d; body = %q", rec.Code, rec.Body.String())
	}

	conflictBody := fmt.Sprintf(`{"name":"Stealer","label_ids":[%d]}`, label.ID)
	rec := santaAdminRequest(t, router, cookie, http.MethodPost, "/api/santa/configurations", conflictBody)
	if rec.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d, want %d; body = %q", rec.Code, http.StatusConflict, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"configuration_name":"Owner"`) {
		t.Fatalf("conflict response = %q, want configuration_name=Owner", rec.Body.String())
	}
}

func TestSantaEventsListFiltersAndPaginates(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	santaStore := santa.NewStore(db)
	eventsStore := santaevents.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-events-wire-host",
		OrbitNodeKey: "santa-events-wire-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := santaStore.UpsertHostObservation(ctx, santa.HostObservation{
		HostID:             host.ID,
		MachineID:          "santa-events-wire-host",
		SerialNumber:       "WIRE",
		ClientModeReported: configurations.ReportedClientModeMonitor,
	}); err != nil {
		t.Fatalf("upsert observation: %v", err)
	}
	occurredAt := time.Date(2026, 5, 23, 14, 0, 0, 0, time.UTC)
	if err := eventsStore.IngestEvents(ctx, host.ID, []santaevents.ExecutionEventInput{
		{
			FileSHA256: "wire-blocked-1",
			FileName:   "Blocked One",
			OccurredAt: occurredAt,
			Decision:   santaevents.ExecutionDecisionBlockBinary,
		},
		{
			FileSHA256: "wire-blocked-2",
			FileName:   "Blocked Two",
			OccurredAt: occurredAt.Add(time.Second),
			Decision:   santaevents.ExecutionDecisionBlockCertificate,
		},
		{
			FileSHA256: "wire-allowed",
			FileName:   "Allowed",
			OccurredAt: occurredAt.Add(2 * time.Second),
			Decision:   santaevents.ExecutionDecisionAllowBinary,
		},
	}, []santaevents.FileAccessEventInput{
		{
			RuleVersion: "wire-v1",
			RuleName:    "Protect Wire Payroll",
			Target:      "/Users/alice/WirePayroll.csv",
			Decision:    santaevents.FileAccessDecisionDenied,
			OccurredAt:  occurredAt.Add(3 * time.Second),
			ProcessChain: []santaevents.ProcessInput{{
				PID:        42,
				FilePath:   "/Applications/Wire.app/Contents/MacOS/Wire",
				FileSHA256: "wire-process",
			}},
		},
		{
			RuleVersion: "wire-v1",
			RuleName:    "Audit Downloads",
			Target:      "/Users/alice/Downloads/audit.txt",
			Decision:    santaevents.FileAccessDecisionAuditOnly,
			OccurredAt:  occurredAt.Add(4 * time.Second),
		},
	}); err != nil {
		t.Fatalf("ingest events: %v", err)
	}

	router, protected, cookie := santaAdminTestAPI(t, db, "events-wire-admin@example.test")
	RegisterSantaEvents(protected, eventsStore)

	rec := santaAdminRequest(
		t,
		router,
		cookie,
		http.MethodGet,
		"/api/santa/events?decisions=blocked&page_size=1",
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Blocked") {
		t.Fatalf("body = %q, want a blocked event", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "Allowed") {
		t.Fatalf("body = %q, decisions=blocked filter did not exclude allowed events", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"count":2`) {
		t.Fatalf("body = %q, want count=2 for normal pagination", rec.Body.String())
	}

	rec = santaAdminRequest(t, router, cookie, http.MethodGet, "/api/santa/events?q=Allowed&decisions=allowed", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Allowed") || strings.Contains(rec.Body.String(), "Blocked") {
		t.Fatalf("search response = %q, want only allowed event", rec.Body.String())
	}

	rec = santaAdminRequest(t, router, cookie, http.MethodGet, "/api/santa/file-access-events?decisions=denied", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("file access status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var fileAccessList paginatedBody[santaevents.FileAccessEvent]
	if err := json.Unmarshal(rec.Body.Bytes(), &fileAccessList); err != nil {
		t.Fatalf("decode file access list: %v", err)
	}
	if fileAccessList.Count != 1 ||
		len(fileAccessList.Items) != 1 ||
		fileAccessList.Items[0].Decision != santaevents.FileAccessDecisionDenied ||
		fileAccessList.Items[0].Target != "/Users/alice/WirePayroll.csv" {
		t.Fatalf("file access list = %+v, want one denied payroll event", fileAccessList)
	}

	rec = santaAdminRequest(
		t,
		router,
		cookie,
		http.MethodGet,
		fmt.Sprintf("/api/santa/file-access-events/%d", fileAccessList.Items[0].ID),
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("file access detail status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var fileAccessDetail santaevents.FileAccessEvent
	if err := json.Unmarshal(rec.Body.Bytes(), &fileAccessDetail); err != nil {
		t.Fatalf("decode file access detail: %v", err)
	}
	if len(fileAccessDetail.ProcessChain) != 1 || fileAccessDetail.ProcessChain[0].FileName != "Wire" {
		t.Fatalf("file access detail = %+v, want persisted process chain", fileAccessDetail)
	}
}

func TestHostDetailRunsSantaEnricher(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	santaStore := santa.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID:   "santa-enricher-host",
		HardwareSerial: "ENRICH",
		OrbitNodeKey:   "santa-enricher-orbit",
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
	label, err := labels.NewStore(db).Create(ctx, labels.LabelCreate{
		Name:                "Enricher Label",
		LabelType:           labels.LabelTypeRegular,
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	if err := labels.NewStore(db).SetMembership(ctx, label.ID, host.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}
	configuration, err := configurations.NewStore(db).CreateConfiguration(ctx, configurations.ConfigurationMutation{
		Name:     "Enricher Config",
		LabelIDs: []int64{label.ID},
	})
	if err != nil {
		t.Fatalf("create configuration: %v", err)
	}

	hostState := santa.NewHostStateService(santaStore, configurations.NewStore(db))
	router, cookie := santaAuthedRouter(t, db, "enricher-admin@example.test", func(api huma.API) {
		RegisterHosts(api, hostStore, nil, SantaHostDetailContributor(hostState))
	})
	rec := santaAdminRequest(t, router, cookie, http.MethodGet, fmt.Sprintf("/api/hosts/%d", host.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Santa *struct {
			Version                string `json:"version"`
			ClientModeReported     string `json:"client_mode_reported"`
			EffectiveConfiguration *struct {
				ID              int64 `json:"id"`
				MatchedViaLabel *struct {
					ID int64 `json:"id"`
				} `json:"matched_via_label"`
			} `json:"effective_configuration"`
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
	if body.Santa.EffectiveConfiguration == nil ||
		body.Santa.EffectiveConfiguration.ID != configuration.ID ||
		body.Santa.EffectiveConfiguration.MatchedViaLabel == nil ||
		body.Santa.EffectiveConfiguration.MatchedViaLabel.ID != label.ID {
		t.Fatalf("effective configuration = %+v, want id=%d via label=%d",
			body.Santa.EffectiveConfiguration, configuration.ID, label.ID)
	}
}

func santaAdminTestAPI(t *testing.T, db *database.DB, email string) (*chi.Mux, huma.API, *http.Cookie) {
	t.Helper()
	return santaTestAPIWith(t, db, email, true)
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

	userService := users.NewService(users.NewStore(db))
	if _, err := userService.Create(context.Background(), users.CreateParams{
		Email:    email,
		Name:     "Santa Test Admin",
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
	if requireAdminGroup {
		protected.UseMiddleware(RequireAdmin(api))
	}

	return router, protected, loginSantaTestUser(t, authService, sessionManager, email)
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
