package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
)

func TestDeleteHostsDecodesCollectionIDs(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := hosts.NewStore(database)
	seeded := make([]*hosts.Host, 0, 3)
	for _, name := range []string{"delete-host-a", "delete-host-b", "keep-host"} {
		host, err := store.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
			Hardware:     hosts.HostHardware{UUID: name},
			OrbitNodeKey: name + "-node-key",
		})
		if err != nil {
			t.Fatalf("enroll %s: %v", name, err)
		}
		seeded = append(seeded, host)
	}

	router := hostTestRouter(t, func(api huma.API) {
		RegisterHosts(api, store, nil, discardLogger())
	})
	for _, path := range []string{"/api/hosts", "/api/hosts?ids="} {
		rec := hostAPIRequest(t, router, http.MethodDelete, path, "")
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf(
				"DELETE %s status = %d, want %d; body = %q",
				path,
				rec.Code,
				http.StatusUnprocessableEntity,
				rec.Body.String(),
			)
		}
		for _, host := range seeded {
			if _, err := store.GetByID(ctx, host.ID); err != nil {
				t.Fatalf("host %d after rejected DELETE: %v", host.ID, err)
			}
		}
	}

	path := fmt.Sprintf("/api/hosts?ids=%d,%d", seeded[0].ID, seeded[1].ID)
	rec := hostAPIRequest(t, router, http.MethodDelete, path, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want %d; body = %q", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	for _, host := range seeded[:2] {
		if _, err := store.GetByID(ctx, host.ID); !errors.Is(err, dbutil.ErrNotFound) {
			t.Fatalf("deleted host %d error = %v, want ErrNotFound", host.ID, err)
		}
	}
	if _, err := store.GetByID(ctx, seeded[2].ID); err != nil {
		t.Fatalf("unselected host %d: %v", seeded[2].ID, err)
	}
}

func TestHostPrimaryUserMutationsRefreshDerivedLabels(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	primaryUserStore := hosts.NewPrimaryUserStore(db)
	labelStore := labels.NewStore(db)
	primaryUsers := primaryUserStore

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "host-manual-user-map"},
		OrbitNodeKey: "host-manual-user-map-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := primaryUserStore.Upsert(
		ctx,
		host.ID,
		"agent@example.test",
		hosts.PrimaryUserSourceOrbitProfile,
	); err != nil {
		t.Fatalf("seed orbit primary user: %v", err)
	}
	var manualUserID int64
	if err := db.Pool().QueryRow(ctx, `
INSERT INTO users (email, name, source, external_id, user_principal_name)
VALUES ('manual@example.test', 'Manual User', 'entra', 'manual-user', 'manual@example.test')
RETURNING id`).Scan(&manualUserID); err != nil {
		t.Fatalf("insert manual directory user: %v", err)
	}
	derivedLabel, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Manual primary user",
		LabelMembershipType: labels.LabelMembershipTypeDerived,
		Criteria: &labels.Criteria{
			Attribute: labels.DerivedAttributeUser,
			Values:    []string{strconv.FormatInt(manualUserID, 10)},
		},
	})
	if err != nil {
		t.Fatalf("create derived label: %v", err)
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
	assertHostLabel(t, ctx, labelStore, host.ID, derivedLabel.ID, true)

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
	assertHostLabel(t, ctx, labelStore, host.ID, derivedLabel.ID, false)
}

func assertHostLabel(
	t *testing.T,
	ctx context.Context,
	store *labels.Store,
	hostID int64,
	labelID int64,
	want bool,
) {
	t.Helper()
	hostLabels, err := store.ListForHost(ctx, hostID)
	if err != nil {
		t.Fatalf("list host labels: %v", err)
	}
	for _, label := range hostLabels {
		if label.ID == labelID {
			if !want {
				t.Fatalf("host %d unexpectedly has label %d", hostID, labelID)
			}
			return
		}
	}
	if want {
		t.Fatalf("host %d does not have label %d", hostID, labelID)
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
