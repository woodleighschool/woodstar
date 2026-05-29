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

func TestHostDeviceMappingManualOverride(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	deviceMappings := hosts.NewDeviceMappingStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "host-manual-user-map",
		OrbitNodeKey: "host-manual-user-map-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := deviceMappings.Upsert(
		ctx,
		host.ID,
		"agent@example.test",
		hosts.DeviceMappingSourceOrbitProfile,
	); err != nil {
		t.Fatalf("seed orbit mapping: %v", err)
	}

	router, cookie := santaAuthedRouter(t, db, "host-mapping-admin@example.test", func(api huma.API) {
		RegisterHosts(api, hostStore, deviceMappings, nil)
	})
	rec := santaAdminRequest(
		t,
		router,
		cookie,
		http.MethodPut,
		fmt.Sprintf("/api/hosts/%d/device-mapping", host.ID),
		`{"email":"manual@example.test"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		DeviceMappings []struct {
			Email  string `json:"email"`
			Source string `json:"source"`
		} `json:"device_mappings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host detail: %v", err)
	}
	if len(body.DeviceMappings) != 2 ||
		body.DeviceMappings[0].Email != "manual@example.test" ||
		body.DeviceMappings[0].Source != string(hosts.DeviceMappingSourceManual) {
		t.Fatalf("device mappings after put = %+v, want manual mapping first", body.DeviceMappings)
	}

	rec = santaAdminRequest(
		t,
		router,
		cookie,
		http.MethodDelete,
		fmt.Sprintf("/api/hosts/%d/device-mapping", host.ID),
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body.DeviceMappings = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host detail after delete: %v", err)
	}
	if len(body.DeviceMappings) != 1 || body.DeviceMappings[0].Email != "agent@example.test" {
		t.Fatalf("device mappings after delete = %+v, want agent mapping only", body.DeviceMappings)
	}
}
