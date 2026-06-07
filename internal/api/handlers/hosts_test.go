package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
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
		RegisterHosts(api, hostStore, userAffinities, nil, checks.NewStore(db))
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
	checkStore := checks.NewStore(db)

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

	check, err := checkStore.Create(ctx, checks.CheckMutation{Name: "Filter Test Check", Query: "select 1"})
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	passes := true
	if err := checkStore.UpsertMembership(ctx, check.ID, passingHost.ID, &passes); err != nil {
		t.Fatalf("upsert passing membership: %v", err)
	}
	fails := false
	if err := checkStore.UpsertMembership(ctx, check.ID, failingHost.ID, &fails); err != nil {
		t.Fatalf("upsert failing membership: %v", err)
	}
	if err := checkStore.UpsertMembership(ctx, check.ID, unevaluatedHost.ID, nil); err != nil {
		t.Fatalf("upsert unevaluated membership: %v", err)
	}

	router, cookie := santaAuthedRouter(t, db, "host-check-filter-admin@example.test", func(api huma.API) {
		RegisterHosts(api, hostStore, hosts.NewUserAffinityStore(db), nil, checkStore)
	})

	tests := []struct {
		response string
		wantID   int64
	}{
		{response: string(checks.CheckStatusPass), wantID: passingHost.ID},
		{response: string(checks.CheckStatusFail), wantID: failingHost.ID},
	}
	for _, tt := range tests {
		path := fmt.Sprintf("/api/hosts?check_id=%d&check_response=%s", check.ID, tt.response)
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

	path := fmt.Sprintf(
		"/api/hosts?ids=%d&ids=%d&check_id=%d&check_response=pass",
		passingHost.ID,
		failingHost.ID,
		check.ID,
	)
	rec := santaAdminRequest(t, router, cookie, http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status = %d, want %d; body = %q", path, rec.Code, http.StatusOK, rec.Body.String())
	}
	var intersected Page[hosts.Host]
	if err := json.Unmarshal(rec.Body.Bytes(), &intersected); err != nil {
		t.Fatalf("decode intersected hosts body: %v", err)
	}
	if intersected.Count != 1 || len(intersected.Items) != 1 || intersected.Items[0].ID != passingHost.ID {
		t.Fatalf("%s hosts = %+v count=%d, want passing host only", path, intersected.Items, intersected.Count)
	}

	path = fmt.Sprintf(
		"/api/hosts?ids=%d&check_id=%d&check_response=pass",
		failingHost.ID,
		check.ID,
	)
	rec = santaAdminRequest(t, router, cookie, http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status = %d, want %d; body = %q", path, rec.Code, http.StatusOK, rec.Body.String())
	}
	var empty Page[hosts.Host]
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil {
		t.Fatalf("decode empty hosts body: %v", err)
	}
	if empty.Count != 0 || len(empty.Items) != 0 {
		t.Fatalf("%s hosts = %+v count=%d, want no hosts", path, empty.Items, empty.Count)
	}

	badPaths := []struct {
		path string
		want int
	}{
		{path: fmt.Sprintf("/api/hosts?check_id=%d", check.ID), want: http.StatusBadRequest},
		{path: "/api/hosts?check_response=pass", want: http.StatusBadRequest},
		{
			path: fmt.Sprintf("/api/hosts?check_id=%d&check_response=unknown", check.ID),
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
