//go:build postgres

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

func TestMunkiPackageMutationLifecycle(t *testing.T) {
	fixture := newMunkiFixture(t)
	rec := fixture.requestJSON(t, http.MethodPost, munkiPackagePath, json.RawMessage(fmt.Sprintf(`{
		"software_id":%d, "version":"  1.0  ", "installer_type":"nopkg",
		"minimum_os_version":"  14.0  ", "notes":"Keep notes", "unattended_install":true,
		"preinstall_alert":{"enabled":true,"title":"Keep title","detail":"Original detail"}
	}`, fixture.softwareID)))
	assertStatus(t, rec, http.StatusCreated, "create package")
	var created packages.Package
	decodeJSON(t, rec, &created)
	if created.Version != "1.0" || created.MinimumOSVersion != "14.0" || !created.UnattendedInstall || created.PreinstallAlert.Title != "Keep title" {
		t.Fatalf("created package = %+v", created)
	}
	path := fmt.Sprintf("%s/%d", munkiPackagePath, created.ID)
	for _, contentType := range []string{"application/json", "application/merge-patch+json"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, path, strings.NewReader(`{"minimum_os_version":"  15.0  ","preinstall_alert":{"detail":"New detail"}}`))
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		fixture.router.ServeHTTP(rec, req)
		assertStatus(t, rec, http.StatusOK, "merge package as "+contentType)
		var merged packages.Package
		decodeJSON(t, rec, &merged)
		if merged.Version != created.Version || merged.Notes != created.Notes || !merged.UnattendedInstall || merged.MinimumOSVersion != "15.0" ||
			!merged.PreinstallAlert.Enabled || merged.PreinstallAlert.Title != "Keep title" || merged.PreinstallAlert.Detail != "New detail" {
			t.Fatalf("merge lost omitted package fields: %+v", merged)
		}
	}
	rec = fixture.requestJSON(t, http.MethodPut, path, json.RawMessage(`{"version":"  2.0  ","installer_type":"nopkg","minimum_os_version":"  16.0  "}`))
	assertStatus(t, rec, http.StatusOK, "replace package")
	var replaced packages.Package
	decodeJSON(t, rec, &replaced)
	if replaced.Version != "2.0" || replaced.MinimumOSVersion != "16.0" || replaced.Notes != "" || replaced.UnattendedInstall || replaced.PreinstallAlert != (packages.PackageAlert{}) {
		t.Fatalf("replacement retained omitted editable package fields: %+v", replaced)
	}
	rec = fixture.request(t, http.MethodGet, path)
	assertStatus(t, rec, http.StatusOK, "read package")
	var persisted packages.Package
	decodeJSON(t, rec, &persisted)
	if persisted.Software.ID != fixture.softwareID || persisted.Version != "2.0" || persisted.MinimumOSVersion != "16.0" || persisted.Notes != "" || persisted.UnattendedInstall {
		t.Fatalf("persisted package = %+v", persisted)
	}
}

func TestMunkiPackageRejectsInvalidMutations(t *testing.T) {
	fixture := newMunkiFixture(t)
	pkg, err := fixture.packages.Create(t.Context(), packages.PackageCreateMutation{
		SoftwareID: fixture.softwareID, Version: "1.0", InstallerType: packages.InstallerTypeNoPkg,
	})
	if err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("%s/%d", munkiPackagePath, pkg.ID)
	for _, tc := range []struct {
		name, method, path, body string
		status                   int
	}{
		{"create requires version", http.MethodPost, munkiPackagePath, fmt.Sprintf(`{"software_id":%d,"installer_type":"nopkg"}`, fixture.softwareID), http.StatusUnprocessableEntity},
		{"replace requires version", http.MethodPut, path, `{"installer_type":"nopkg"}`, http.StatusUnprocessableEntity},
		{"create rejects missing installer", http.MethodPost, munkiPackagePath, fmt.Sprintf(`{"software_id":%d,"version":"2.0","installer_type":"pkg","installer_object_id":999999}`, fixture.softwareID), http.StatusNotFound},
		{"replace rejects missing installer", http.MethodPut, path, `{"version":"2.0","installer_type":"pkg","installer_object_id":999999}`, http.StatusNotFound},
		{"merge rejects missing installer", http.MethodPatch, path, `{"installer_type":"pkg","installer_object_id":999999}`, http.StatusNotFound},
		{"merge requires object", http.MethodPatch, path, `null`, http.StatusUnprocessableEntity},
		{"merge rejects incorrect type", http.MethodPatch, path, `{"notes":1}`, http.StatusUnprocessableEntity},
		{"merge rejects field aliases", http.MethodPatch, path, `{"Notes":"alias"}`, http.StatusUnprocessableEntity},
		{"merge rejects null boolean", http.MethodPatch, path, `{"unattended_install":null}`, http.StatusUnprocessableEntity},
		{"merge rejects null list", http.MethodPatch, path, `{"requires":null}`, http.StatusUnprocessableEntity},
		{"merge rejects null object", http.MethodPatch, path, `{"preinstall_alert":null}`, http.StatusUnprocessableEntity},
		{"merge rejects nested null boolean", http.MethodPatch, path, `{"preinstall_alert":{"enabled":null}}`, http.StatusUnprocessableEntity},
		{"replace rejects reparenting", http.MethodPut, path, `{"software_id":1,"version":"2.0","installer_type":"nopkg"}`, http.StatusUnprocessableEntity},
		{"merge rejects reparenting", http.MethodPatch, path, `{"software_id":1}`, http.StatusUnprocessableEntity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := fixture.requestJSON(t, tc.method, tc.path, json.RawMessage(tc.body))
			assertProblem(t, rec, tc.status)
		})
	}
}
