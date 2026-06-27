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

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

func TestSantaConfigurationOverlappingTargetsAreAllowed(t *testing.T) {
	db, ctx := dbtest.Open(t)
	router := chi.NewRouter()
	api := humachi.New(router, huma.DefaultConfig("test", "test"))
	registerSantaConfigurations(api, configurations.NewStore(db), discardLogger())

	label, err := labels.NewStore(db).Create(ctx, labels.LabelMutation{
		Name:                "Conflict Label",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}

	body := santaConfigurationBody("Owner", label.ID)
	if rec := santaConfigurationRequest(
		t,
		router,
		http.MethodPost,
		"/api/santa/configurations",
		body,
	); rec.Code != http.StatusCreated {
		t.Fatalf("seed configuration status = %d; body = %q", rec.Code, rec.Body.String())
	}

	overlapBody := santaConfigurationBody("Second", label.ID)
	rec := santaConfigurationRequest(t, router, http.MethodPost, "/api/santa/configurations", overlapBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("overlap status = %d, want %d; body = %q", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var created configurations.Configuration
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode configuration: %v", err)
	}
	if len(created.Targets.Include) != 1 || created.Targets.Include[0].LabelID != label.ID ||
		len(created.Targets.Exclude) != 0 {
		t.Fatalf("targets = %+v, want canonical include target", created.Targets)
	}

	updateBody := santaConfigurationBody("Second Updated", label.ID)
	rec = santaConfigurationRequest(
		t,
		router,
		http.MethodPut,
		fmt.Sprintf("/api/santa/configurations/%d", created.ID),
		updateBody,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func santaConfigurationBody(name string, labelID int64) string {
	return fmt.Sprintf(`{
		"name": %q,
		"client_mode": "monitor",
		"enable_bundles": false,
		"enable_transitive_rules": false,
		"enable_all_event_upload": false,
		"full_sync_interval_seconds": 600,
		"batch_size": 50,
		"targets": {"include": [{"label_id": %d}], "exclude": []}
	}`, name, labelID)
}

func santaConfigurationRequest(
	t *testing.T,
	router *chi.Mux,
	method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), method, path, http.NoBody)
	if body != "" {
		req = httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(rec, req)
	return rec
}
