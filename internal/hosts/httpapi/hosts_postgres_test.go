//go:build postgres

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/geoip"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/testutil/testbloby"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestDeleteHostsDecodesCollectionIDs(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := hosts.NewStore(database, labels.NewStore(database))
	activities := &recordingActivity{}
	seeded := make([]*hosts.Host, 0, 3)
	for _, name := range []string{"delete-host-a", "delete-host-b", "keep-host"} {
		host, err := store.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
			Hardware:     hosts.HostHardware{UUID: name},
			OrbitNodeKey: name + "-node-key",
		}, heartbeats.Contact{})

		if err != nil {
			t.Fatalf("enroll %s: %v", name, err)
		}
		seeded = append(seeded, host)
	}

	router := hostTestRouter(t, func(humaAPI huma.API) {
		RegisterAPI(api.AppRoutes{Ordinary: humaAPI}, store, nil, nil, nil, nil, nil, activities, discardLogger())
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
		if _, err := store.GetByID(ctx, host.ID); !errors.Is(err, fault.ErrNotFound) {
			t.Fatalf("deleted host %d error = %v, want ErrNotFound", host.ID, err)
		}
	}
	if _, err := store.GetByID(ctx, seeded[2].ID); err != nil {
		t.Fatalf("unselected host %d: %v", seeded[2].ID, err)
	}
	if len(activities.events) != 1 || activities.events[0].Action != activity.ActionHostsDeleted ||
		activities.events[0].Actor.UserID == nil || *activities.events[0].Actor.UserID != 1 ||
		activities.events[0].Subject.Type != hostResource || activities.events[0].Subject.Name != "2 hosts" {
		t.Fatalf("activity = %+v, want one authenticated bulk-delete event", activities.events)
	}
}

func TestHostPrimaryUserMutationsRefreshDerivedLabels(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	hostStore := hosts.NewStore(db, labelStore)
	primaryUserStore := hosts.NewPrimaryUserStore(db, labelStore)
	primaryUsers := primaryUserStore

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "host-manual-user-map"},
		OrbitNodeKey: "host-manual-user-map-orbit",
	}, heartbeats.Contact{})

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
	if err := db.QueryRow(ctx, `
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

	router := hostTestRouter(t, func(humaAPI huma.API) {
		RegisterAPI(
			api.AppRoutes{Ordinary: humaAPI},
			hostStore,
			primaryUsers,
			nil,
			nil,
			nil,
			nil,
			nil,
			discardLogger(),
		)
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

func TestHostResponsesBatchEnrichFlatAgentContract(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db, labels.NewStore(db))
	now := time.Now().UTC().Truncate(time.Microsecond)
	host, err := hostStore.UpsertOnOsqueryEnroll(ctx, hosts.InventoryUpdate{
		Hardware:       hosts.HostHardware{UUID: "host-agent-contract"},
		OsqueryNodeKey: "host-agent-contract-osquery",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := hostStore.ApplyInventory(ctx, host.ID, hosts.InventoryUpdate{LastRestartedAt: &now}); err != nil {
		t.Fatalf("apply inventory: %v", err)
	}
	if err := hostStore.MarkInventoryFresh(ctx, host.ID, "test-inventory-query-hash"); err != nil {
		t.Fatalf("mark inventory fresh: %v", err)
	}
	if _, err := db.Exec(ctx, `
UPDATE host_heartbeats
SET last_seen_at = $2, remote_ip = '198.51.100.40', user_agent = 'osquery/5.14'
WHERE host_id = $1 AND source = 'osquery'`, host.ID, now); err != nil {
		t.Fatalf("update heartbeat: %v", err)
	}

	munkiVersions := &testAgentVersionLoader{versions: map[int64]string{host.ID: "6.6.0"}}
	santaVersions := &testAgentVersionLoader{versions: map[int64]string{host.ID: "2026.4"}}
	router := hostTestRouter(t, func(humaAPI huma.API) {
		RegisterAPI(
			api.AppRoutes{Ordinary: humaAPI},
			hostStore,
			nil,
			munkiVersions,
			santaVersions,
			nil,
			nil,
			nil,
			discardLogger(),
		)
	})

	listRec := hostAPIRequest(t, router, http.MethodGet, "/api/hosts", "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body = %q", listRec.Code, http.StatusOK, listRec.Body.String())
	}
	var list api.Page[hosts.Host]
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode host list: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("list items = %d, want 1", len(list.Items))
	}

	detailRec := hostAPIRequest(t, router, http.MethodGet, fmt.Sprintf("/api/hosts/%d", host.ID), "")
	if detailRec.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d; body = %q", detailRec.Code, http.StatusOK, detailRec.Body.String())
	}
	var detail hosts.HostDetail
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode host detail: %v", err)
	}

	if list.Items[0].Agents.Munki.Version != "6.6.0" || list.Items[0].Agents.Santa.Version != "2026.4" {
		t.Fatalf("list agents = %+v, want Munki and Santa versions", list.Items[0].Agents)
	}
	if detail.Agents.Munki != list.Items[0].Agents.Munki || detail.Agents.Santa != list.Items[0].Agents.Santa {
		t.Fatalf("detail agents = %+v, want list enrichment %+v", detail.Agents, list.Items[0].Agents)
	}
	assertFlatHostContract(t, listRec.Body.Bytes(), true)
	assertFlatHostContract(t, detailRec.Body.Bytes(), false)

	for name, loader := range map[string]*testAgentVersionLoader{
		"munki": munkiVersions,
		"santa": santaVersions,
	} {
		if len(loader.calls) != 2 {
			t.Fatalf("%s AgentVersions calls = %v, want one list and one detail batch", name, loader.calls)
		}
		for _, ids := range loader.calls {
			if len(ids) != 1 || ids[0] != host.ID {
				t.Fatalf("%s AgentVersions ids = %v, want [%d]", name, ids, host.ID)
			}
		}
	}
}

func TestHostResponsesEnrichPublicIP(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db, labels.NewStore(db))
	host, err := hostStore.UpsertOnOsqueryEnroll(ctx, hosts.InventoryUpdate{
		Hardware:       hosts.HostHardware{UUID: "host-public-ip-enrichment"},
		OsqueryNodeKey: "host-public-ip-enrichment-osquery",
	}, heartbeats.Contact{})

	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if _, err := db.Exec(ctx, `
UPDATE host_heartbeats
SET last_seen_at = now(), remote_ip = '198.51.100.40', user_agent = 'osquery/5.14'
WHERE host_id = $1 AND source = 'osquery'`, host.ID); err != nil {
		t.Fatalf("update heartbeat: %v", err)
	}
	distribution := mdp.NewStore(
		db,
		testbloby.New(t, db),
		discardLogger(),
	)
	point, err := distribution.Create(ctx, mdp.DistributionPointMutation{
		Name:          "Senior Campus",
		Enabled:       true,
		ClientCIDRs:   []string{"198.51.100.0/24"},
		ClientBaseURL: "https://mdp.example",
	}, "test-host-enrichment-key")
	if err != nil {
		t.Fatalf("create distribution point: %v", err)
	}
	geo := &testGeoIPLookup{result: &geoip.Result{
		CountryCode:  "AU",
		Country:      "Australia",
		Region:       "Victoria",
		City:         "Langwarrin",
		Latitude:     -38.15,
		Longitude:    145.12,
		ASN:          1221,
		Organization: "Telstra",
	}}
	router := hostTestRouter(t, func(humaAPI huma.API) {
		RegisterAPI(
			api.AppRoutes{Ordinary: humaAPI},
			hostStore,
			nil,
			nil,
			nil,
			distribution,
			geo.Lookup,
			nil,
			discardLogger(),
		)
	})

	listRec := hostAPIRequest(t, router, http.MethodGet, "/api/hosts", "")
	var list api.Page[hosts.Host]
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d; body = %q", listRec.Code, listRec.Body.String())
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode host list: %v", err)
	}
	detailRec := hostAPIRequest(t, router, http.MethodGet, fmt.Sprintf("/api/hosts/%d", host.ID), "")
	var detail hosts.HostDetail
	if detailRec.Code != http.StatusOK {
		t.Fatalf("detail status = %d; body = %q", detailRec.Code, detailRec.Body.String())
	}
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode host detail: %v", err)
	}

	listDetails := list.Items[0].PublicIPDetails
	if listDetails == nil || listDetails.DistributionPoint == nil {
		t.Fatalf("list public IP details = %+v", listDetails)
	}
	if listDetails.DistributionPoint.ID != point.ID ||
		listDetails.DistributionPoint.Name != "Senior Campus" ||
		listDetails.City != "Langwarrin" ||
		listDetails.ASN != 1221 {
		t.Fatalf("list public IP details = %+v", listDetails)
	}
	if !reflect.DeepEqual(detail.PublicIPDetails, listDetails) {
		t.Fatalf("detail public IP details = %+v, want %+v", detail.PublicIPDetails, listDetails)
	}
}

type testGeoIPLookup struct {
	result *geoip.Result
}

func (l *testGeoIPLookup) Lookup(_ netip.Addr) (*geoip.Result, error) {
	return l.result, nil
}

type testAgentVersionLoader struct {
	versions map[int64]string
	calls    [][]int64
}

func (l *testAgentVersionLoader) AgentVersions(_ context.Context, hostIDs []int64) (map[int64]string, error) {
	l.calls = append(l.calls, append([]int64(nil), hostIDs...))
	return l.versions, nil
}

func assertFlatHostContract(t *testing.T, payload []byte, list bool) {
	t.Helper()
	var host map[string]any
	if list {
		var page struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(payload, &page); err != nil {
			t.Fatalf("decode list contract: %v", err)
		}
		host = page.Items[0]
	} else if err := json.Unmarshal(payload, &host); err != nil {
		t.Fatalf("decode detail contract: %v", err)
	}
	for _, key := range []string{
		"public_ip",
		"last_contact",
		"created_at",
		"updated_at",
		"inventory_updated_at",
		"last_restarted_at",
		"heartbeats",
	} {
		if _, ok := host[key]; !ok {
			t.Fatalf("host response missing root %q: %+v", key, host)
		}
	}
	if _, ok := host["timestamps"]; ok {
		t.Fatalf("host response retained nested timestamps: %+v", host["timestamps"])
	}
	network, ok := host["network"].(map[string]any)
	if !ok {
		t.Fatalf("network = %#v, want object", host["network"])
	}
	if _, ok := network["last_remote_ip"]; ok {
		t.Fatalf("network retained last_remote_ip: %+v", network)
	}
	agents, ok := host["agents"].(map[string]any)
	if !ok {
		t.Fatalf("agents = %#v, want object", host["agents"])
	}
	for _, agent := range []string{"orbit", "osquery", "munki", "santa"} {
		if _, ok := agents[agent].(map[string]any); !ok {
			t.Fatalf("agents missing %q object: %+v", agent, agents)
		}
	}
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
	humaAPI := humachi.New(router, cfg)
	protected := huma.NewGroup(humaAPI)
	protected.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		role := directory.RoleAdmin
		next(huma.WithContext(ctx, ctxkeys.WithUser(ctx.Context(), &directory.User{
			ID:    1,
			Email: "host-admin@example.test",
			Role:  &role,
		})))
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

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

type recordingActivity struct {
	events []activity.NewEvent
}

func (recorder *recordingActivity) Record(_ context.Context, event activity.NewEvent) error {
	recorder.events = append(recorder.events, event)
	return nil
}
