//go:build postgres

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/woodleighschool/woodstar/internal/labels"
)

func TestMunkiSoftwareMutationLifecycle(t *testing.T) {
	fixture := newMunkiFixture(t)
	label, err := labels.NewStore(fixture.db).Create(t.Context(), labels.LabelMutation{
		Name: "Excluded devices", LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := fixture.requestJSON(t, http.MethodPost, munkiSoftwarePath, json.RawMessage(fmt.Sprintf(`{
		"name":"LifecycleApp", "display_name":"  Lifecycle App  ",
		"description":"  Keep description  ", "category":"Keep category", "developer":"  Original publisher  ",
		"targets":{"include":[],"exclude":[{"label_id":%d}]}
	}`, label.ID)))
	assertStatus(t, rec, http.StatusCreated, "create software")
	var created munkiSoftwareDetail
	decodeJSON(t, rec, &created)
	if created.DisplayName == nil || *created.DisplayName != "Lifecycle App" || created.Description != "Keep description" || created.Developer != "Original publisher" || len(created.Targets.Exclude) != 1 {
		t.Fatalf("created software = %+v", created)
	}
	path := fmt.Sprintf("%s/%d", munkiSoftwarePath, created.ID)
	rec = fixture.requestJSON(t, http.MethodPatch, path, json.RawMessage(`{"developer":"  Publisher  ","targets":{"include":[]}}`))
	assertStatus(t, rec, http.StatusOK, "merge software")
	var merged munkiSoftwareDetail
	decodeJSON(t, rec, &merged)
	if merged.Name != created.Name || merged.Description != created.Description || merged.Category != created.Category || merged.Developer != "Publisher" ||
		len(merged.Targets.Exclude) != 1 || merged.Targets.Exclude[0].LabelID != label.ID {
		t.Fatalf("merge lost omitted software fields or targets: %+v", merged)
	}
	rec = fixture.requestJSON(t, http.MethodPut, path, json.RawMessage(`{"developer":"  Replacement publisher  ","targets":{"include":[],"exclude":[]}}`))
	assertStatus(t, rec, http.StatusOK, "replace software")
	var replaced munkiSoftwareDetail
	decodeJSON(t, rec, &replaced)
	if replaced.Name != created.Name || replaced.DisplayName != nil || replaced.Description != "" || replaced.Category != "" || replaced.Developer != "Replacement publisher" || len(replaced.Targets.Exclude) != 0 {
		t.Fatalf("replacement retained omitted editable software fields: %+v", replaced)
	}
	rec = fixture.requestJSON(t, http.MethodPatch, path, json.RawMessage(`{"developer":null}`))
	assertStatus(t, rec, http.StatusOK, "clear developer")
	rec = fixture.request(t, http.MethodGet, path)
	assertStatus(t, rec, http.StatusOK, "read software")
	var persisted munkiSoftwareDetail
	decodeJSON(t, rec, &persisted)
	if persisted.Name != created.Name || persisted.Developer != "" || persisted.Description != "" || len(persisted.Targets.Exclude) != 0 {
		t.Fatalf("persisted software = %+v", persisted)
	}
}

func TestMunkiSoftwareRejectsInvalidMutations(t *testing.T) {
	fixture := newMunkiFixture(t)
	path := fmt.Sprintf("%s/%d", munkiSoftwarePath, fixture.softwareID)
	for _, tc := range []struct {
		name, method, path, body string
		status                   int
	}{
		{"create requires name", http.MethodPost, munkiSoftwarePath, `{"targets":{"include":[],"exclude":[]}}`, http.StatusUnprocessableEntity},
		{"create rejects missing icon", http.MethodPost, munkiSoftwarePath, `{"name":"MissingIcon","icon_object_id":999999,"targets":{"include":[],"exclude":[]}}`, http.StatusNotFound},
		{"replace rejects missing icon", http.MethodPut, path, `{"icon_object_id":999999,"targets":{"include":[],"exclude":[]}}`, http.StatusNotFound},
		{"merge rejects missing icon", http.MethodPatch, path, `{"icon_object_id":999999}`, http.StatusNotFound},
		{"replace rejects rename", http.MethodPut, path, `{"name":"rename","targets":{"include":[],"exclude":[]}}`, http.StatusUnprocessableEntity},
		{"merge rejects rename", http.MethodPatch, path, `{"name":"rename"}`, http.StatusUnprocessableEntity},
		{"merge rejects null targets", http.MethodPatch, path, `{"targets":null}`, http.StatusUnprocessableEntity},
		{"merge rejects null includes", http.MethodPatch, path, `{"targets":{"include":null}}`, http.StatusUnprocessableEntity},
		{"merge requires complete list items", http.MethodPatch, path, `{"targets":{"include":[{}]}}`, http.StatusUnprocessableEntity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := fixture.requestJSON(t, tc.method, tc.path, json.RawMessage(tc.body))
			assertProblem(t, rec, tc.status)
		})
	}
}
