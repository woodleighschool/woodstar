package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

func TestHostPrimaryUserManualOverride(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	primaryUsers := hosts.NewPrimaryUserStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "host-manual-user-map"},
		OrbitNodeKey: "host-manual-user-map-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := primaryUsers.Upsert(
		ctx,
		host.ID,
		"agent@example.test",
		hosts.PrimaryUserSourceOrbitProfile,
	); err != nil {
		t.Fatalf("seed orbit primary user: %v", err)
	}

	router := hostTestRouter(t, func(api huma.API) {
		RegisterHosts(api, hostStore, primaryUsers, discardLogger())
	})
	rec := hostAPIRequest(
		t,
		router,
		http.MethodPut,
		fmt.Sprintf("/api/hosts/%d/primary-user", host.ID),
		`{"email":"manual@example.test"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		PrimaryUserSources []struct {
			Email  string `json:"email"`
			Source string `json:"source"`
		} `json:"primary_user_sources"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host detail: %v", err)
	}
	if len(body.PrimaryUserSources) != 2 ||
		body.PrimaryUserSources[0].Email != "manual@example.test" ||
		body.PrimaryUserSources[0].Source != string(hosts.PrimaryUserSourceManual) {
		t.Fatalf("primary user sources after put = %+v, want manual source first", body.PrimaryUserSources)
	}

	rec = hostAPIRequest(
		t,
		router,
		http.MethodDelete,
		fmt.Sprintf("/api/hosts/%d/primary-user", host.ID),
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body.PrimaryUserSources = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host detail after delete: %v", err)
	}
	if len(body.PrimaryUserSources) != 1 || body.PrimaryUserSources[0].Email != "agent@example.test" {
		t.Fatalf("primary user sources after delete = %+v, want agent source only", body.PrimaryUserSources)
	}
}

func hostTestRouter(t *testing.T, register func(huma.API)) *chi.Mux {
	t.Helper()

	router := chi.NewRouter()
	cfg := huma.DefaultConfig("test", "test")
	cfg.Components = &huma.Components{
		Schemas: huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer),
	}
	api := humachi.New(router, cfg)
	protected := huma.NewGroup(api)
	protected.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		adminRole := directory.RoleAdmin
		user := &directory.User{ID: 1, Email: "host-admin@example.test", Role: &adminRole}
		next(huma.WithContext(ctx, ctxkeys.WithUser(ctx.Context(), user)))
	})
	register(protected)
	return router
}

func hostAPIRequest(t *testing.T, router *chi.Mux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(rec, req)
	return rec
}
