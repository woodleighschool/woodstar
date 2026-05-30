package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

func TestHostUserAffinityManualOverride(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	userAffinities := hosts.NewUserAffinityStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "host-manual-user-map"},
		OrbitNodeKey: "host-manual-user-map-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := userAffinities.Upsert(
		ctx,
		host.ID,
		"agent@example.test",
		hosts.UserAffinitySourceOrbitProfile,
	); err != nil {
		t.Fatalf("seed orbit mapping: %v", err)
	}

	router, cookie := santaAuthedRouter(t, db, "host-mapping-admin@example.test", func(api huma.API) {
		RegisterHosts(api, hostStore, userAffinities, nil)
	})
	rec := santaAdminRequest(
		t,
		router,
		cookie,
		http.MethodPut,
		fmt.Sprintf("/api/hosts/%d/user-affinity", host.ID),
		`{"email":"manual@example.test"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		UserAffinity struct {
			Mappings []struct {
				Email  string `json:"email"`
				Source string `json:"source"`
			} `json:"mappings"`
		} `json:"user_affinity"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host detail: %v", err)
	}
	if len(body.UserAffinity.Mappings) != 2 ||
		body.UserAffinity.Mappings[0].Email != "manual@example.test" ||
		body.UserAffinity.Mappings[0].Source != string(hosts.UserAffinitySourceManual) {
		t.Fatalf("user affinity mappings after put = %+v, want manual mapping first", body.UserAffinity.Mappings)
	}

	rec = santaAdminRequest(
		t,
		router,
		cookie,
		http.MethodDelete,
		fmt.Sprintf("/api/hosts/%d/user-affinity", host.ID),
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body.UserAffinity.Mappings = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host detail after delete: %v", err)
	}
	if len(body.UserAffinity.Mappings) != 1 || body.UserAffinity.Mappings[0].Email != "agent@example.test" {
		t.Fatalf("user affinity mappings after delete = %+v, want agent mapping only", body.UserAffinity.Mappings)
	}
}

func TestHostListCheckResponseFilter(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)

	passingHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "host-check-filter-passing"},
		OrbitNodeKey: "host-check-filter-passing-orbit",
	})
	if err != nil {
		t.Fatalf("enroll passing host: %v", err)
	}
	failingHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "host-check-filter-failing"},
		OrbitNodeKey: "host-check-filter-failing-orbit",
	})
	if err != nil {
		t.Fatalf("enroll failing host: %v", err)
	}
	unevaluatedHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "host-check-filter-unevaluated"},
		OrbitNodeKey: "host-check-filter-unevaluated-orbit",
	})
	if err != nil {
		t.Fatalf("enroll unevaluated host: %v", err)
	}

	var checkID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO checks (name, query)
		VALUES ('Filter Test Check', 'select 1')
		RETURNING id
	`).Scan(&checkID); err != nil {
		t.Fatalf("insert check: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO check_membership (check_id, host_id, passes)
		VALUES ($1, $2, TRUE), ($1, $3, FALSE), ($1, $4, NULL)
	`, checkID, passingHost.ID, failingHost.ID, unevaluatedHost.ID); err != nil {
		t.Fatalf("insert check membership: %v", err)
	}

	router, cookie := santaAuthedRouter(t, db, "host-check-filter-admin@example.test", func(api huma.API) {
		RegisterHosts(api, hostStore, hosts.NewUserAffinityStore(db), nil)
	})

	tests := []struct {
		response string
		wantID   int64
	}{
		{response: string(hosts.CheckResponsePass), wantID: passingHost.ID},
		{response: string(hosts.CheckResponseFail), wantID: failingHost.ID},
	}
	for _, tt := range tests {
		path := fmt.Sprintf("/api/hosts?check_id=%d&check_response=%s", checkID, tt.response)
		rec := santaAdminRequest(t, router, cookie, http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d; body = %q", path, rec.Code, http.StatusOK, rec.Body.String())
		}
		var body Page[hosts.Host]
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode hosts body: %v", err)
		}
		if body.Count != 1 || len(body.Items) != 1 || body.Items[0].ID != tt.wantID {
			t.Fatalf("%s hosts = %+v count=%d, want host %d only", path, body.Items, body.Count, tt.wantID)
		}
	}

	badPaths := []struct {
		path string
		want int
	}{
		{path: fmt.Sprintf("/api/hosts?check_id=%d", checkID), want: http.StatusBadRequest},
		{path: "/api/hosts?check_response=pass", want: http.StatusBadRequest},
		{
			path: fmt.Sprintf("/api/hosts?check_id=%d&check_response=unknown", checkID),
			want: http.StatusUnprocessableEntity,
		},
	}
	for _, tt := range badPaths {
		rec := santaAdminRequest(t, router, cookie, http.MethodGet, tt.path, "")
		if rec.Code != tt.want {
			t.Fatalf("%s status = %d, want %d; body = %q", tt.path, rec.Code, tt.want, rec.Body.String())
		}
	}
}
