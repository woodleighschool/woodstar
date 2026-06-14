package rules_test

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

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
)

func TestSantaRuleReferencesEndpointReturnsCandidates(t *testing.T) {
	db, ctx := dbtest.Open(t)
	router := santaRulesAPI(t, func(api huma.API) {
		rules.RegisterAdminRoutes(api, rules.NewStore(db))
	})

	identifier := strings.Repeat("5", 64)
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_executables (sha256, file_name, file_bundle_id, team_id)
		VALUES ($1, 'Endpoint Target', 'com.example.endpoint', 'TEAMENDPT')
	`, identifier); err != nil {
		t.Fatalf("insert executable: %v", err)
	}

	rec := santaRulesRequest(
		t,
		router,
		http.MethodGet,
		"/api/santa/rule-references?rule_type=binary&q=Endpoint",
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body []rules.RuleReferenceCandidate
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body) != 1 || body[0].Identifier != identifier {
		t.Fatalf("references = %+v, want endpoint executable", body)
	}
	if body[0].DisplayName != "" ||
		body[0].FileName != "Endpoint Target" ||
		body[0].BundleIdentifier != "com.example.endpoint" {
		t.Fatalf("reference metadata = %+v, want semantic executable fields", body[0])
	}
}

func TestSantaRuleEndpointCreatesSigningIDWithoutTargets(t *testing.T) {
	db, _ := dbtest.Open(t)
	router := santaRulesAPI(t, func(api huma.API) {
		rules.RegisterAdminRoutes(api, rules.NewStore(db))
	})

	rec := santaRulesRequest(
		t,
		router,
		http.MethodPost,
		"/api/santa/rules",
		`{
			"name": "Google Chrome Signing ID",
			"rule_type": "signingid",
			"identifier": "EQHXZ8M8AV:com.google.Chrome",
			"targets": {
				"include": [],
				"exclude": []
			}
		}`,
	)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %q", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var created rules.Rule
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created rule: %v", err)
	}
	if created.RuleType != rules.RuleTypeSigningID ||
		created.Identifier != "EQHXZ8M8AV:com.google.Chrome" ||
		created.Name != "Google Chrome Signing ID" {
		t.Fatalf("created rule = %+v, want Chrome signing ID rule", created)
	}
	if len(created.Targets.Include) != 0 || len(created.Targets.Exclude) != 0 {
		t.Fatalf("created targets = %+v, want empty targets", created.Targets)
	}
}

func TestSantaRuleEndpointReplacesTargetsOnPut(t *testing.T) {
	db, ctx := dbtest.Open(t)
	router := santaRulesAPI(t, func(api huma.API) {
		rules.RegisterAdminRoutes(api, rules.NewStore(db))
	})

	labelStore := labels.NewStore(db)
	firstLabel, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Rule Endpoint First",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create first label: %v", err)
	}
	secondLabel, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Rule Endpoint Second",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create second label: %v", err)
	}
	excludeLabel, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Rule Endpoint Exclude",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create exclude label: %v", err)
	}

	createBody := santaRuleBody(
		"Endpoint Rule",
		string(rules.RuleTypeBinary),
		strings.Repeat("6", 64),
		string(rules.PolicyAllowlist),
		firstLabel.ID,
		"",
	)
	rec := santaRulesRequest(t, router, http.MethodPost, "/api/santa/rules", createBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %q", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var created rules.Rule
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created rule: %v", err)
	}
	if len(created.Targets.Include) != 1 ||
		created.Targets.Include[0].LabelID != firstLabel.ID ||
		created.Targets.Include[0].Policy != rules.PolicyAllowlist ||
		len(created.Targets.Exclude) != 0 {
		t.Fatalf("created targets = %+v, want canonical include-only target set", created.Targets)
	}

	celExpression := "target.path.startsWith('/Applications')"
	updateBody := santaRuleBody(
		"Endpoint Rule Updated",
		string(rules.RuleTypeSigningID),
		"ABCDE12345:com.example.updated",
		string(rules.PolicyCEL),
		secondLabel.ID,
		celExpression,
		excludeLabel.ID,
	)
	rec = santaRulesRequest(
		t,
		router,
		http.MethodPut,
		fmt.Sprintf("/api/santa/rules/%d", created.ID),
		updateBody,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var updated rules.Rule
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated rule: %v", err)
	}
	if updated.RuleType != rules.RuleTypeSigningID || updated.Identifier != "ABCDE12345:com.example.updated" {
		t.Fatalf("updated identity = %s %q, want signing ID", updated.RuleType, updated.Identifier)
	}
	if len(updated.Targets.Include) != 1 ||
		updated.Targets.Include[0].LabelID != secondLabel.ID ||
		updated.Targets.Include[0].Policy != rules.PolicyCEL ||
		updated.Targets.Include[0].CELExpression != celExpression ||
		len(updated.Targets.Exclude) != 1 ||
		updated.Targets.Exclude[0].LabelID != excludeLabel.ID {
		t.Fatalf("updated targets = %+v, want replaced include/exclude target set", updated.Targets)
	}
}

func TestHostSantaRulesEndpointChecksHostBeforeListingRules(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	ruleStore := rules.NewStore(db)
	router := santaRulesAPI(t, func(api huma.API) {
		rules.RegisterHostAdminRoutes(api, ruleStore, hostStore)
	})

	rec := santaRulesRequest(t, router, http.MethodGet, "/api/hosts/999999/santa/rules", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing host status = %d, want %d; body = %q", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "host-santa-rules-api"},
		OrbitNodeKey: "host-santa-rules-api-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	rec = santaRulesRequest(
		t,
		router,
		http.MethodGet,
		fmt.Sprintf("/api/hosts/%d/santa/rules", host.ID),
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("host rules status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apitypes.Page[rules.RuleStatus]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host rules body: %v", err)
	}
	if body.Count != 0 || len(body.Items) != 0 {
		t.Fatalf("host rules = %+v count=%d, want empty page", body.Items, body.Count)
	}
}

func santaRuleBody(
	name string,
	ruleType string,
	identifier string,
	policy string,
	includeLabelID int64,
	celExpression string,
	excludeLabelIDs ...int64,
) string {
	celField := ""
	if celExpression != "" {
		celField = fmt.Sprintf(`, "cel_expression": %q`, celExpression)
	}
	excludeRows := make([]string, len(excludeLabelIDs))
	for i, labelID := range excludeLabelIDs {
		excludeRows[i] = fmt.Sprintf(`{"label_id": %d}`, labelID)
	}
	return fmt.Sprintf(`{
			"name": %q,
			"rule_type": %q,
			"identifier": %q,
			"targets": {
				"include": [{"label_id": %d, "policy": %q%s}],
				"exclude": [%s]
			}
		}`, name, ruleType, identifier, includeLabelID, policy, celField, strings.Join(excludeRows, ","))
}

func santaRulesAPI(t *testing.T, register func(huma.API)) *chi.Mux {
	t.Helper()

	router := chi.NewRouter()
	api := humachi.New(router, huma.DefaultConfig("test", "test"))
	register(api)
	return router
}

func santaRulesRequest(t *testing.T, router *chi.Mux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(rec, req)
	return rec
}
