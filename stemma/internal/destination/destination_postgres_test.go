//go:build postgres

package destination

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/woodleighschool/goodies/auth/authn"
	"github.com/woodleighschool/goodies/auth/authz"
	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/httpapi"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/testutil/testbloby"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestStemmaAdapterPostgresLifecycle(t *testing.T) { //nolint:funlen,gocognit,cyclop // Sequential ownership and convergence assertions share one real API lifecycle.
	db, ctx := testdb.Open(t)
	objects := testbloby.New(t, db)
	packageStore := packages.NewStore(db, objects)
	softwareStore := software.NewStore(db, objects, packageStore)
	service := munki.NewPackageService(munki.PackageServiceDependencies{
		Packages: packageStore, DesiredPackagesChanged: func() {},
	})
	dependency, err := softwareStore.Create(ctx, software.CreateMutation{Name: "AdapterDependency"})
	if err != nil {
		t.Fatal(err)
	}
	dependencyPackage, err := packageStore.Create(ctx, packages.PackageCreateMutation{
		SoftwareID: dependency.ID, Version: "2.0", InstallerType: packages.InstallerTypeNoPkg,
	})
	if err != nil {
		t.Fatal(err)
	}
	labelStore := labels.NewStore(db)
	included, err := labelStore.Create(ctx, labels.LabelMutation{Name: "Adapter included", LabelMembershipType: labels.LabelMembershipTypeManual})
	if err != nil {
		t.Fatal(err)
	}
	excluded, err := labelStore.Create(ctx, labels.LabelMutation{Name: "Adapter excluded", LabelMembershipType: labels.LabelMembershipTypeManual})
	if err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	cfg := huma.DefaultConfig("test", "test")
	cfg.OpenAPIPath, cfg.DocsPath, cfg.SchemasPath = "", "", ""
	cfg.Components = &huma.Components{Schemas: huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)}
	humaAPI := humachi.New(router, cfg)
	httpapi.RegisterAPI(api.AppRoutes{Protected: humaAPI, LongRunning: humaAPI, Router: router, Transfers: router}, httpapi.Dependencies{
		Software: softwareStore, Packages: service, Objects: objects,
		Authorizer: testAuthorizer{}, Logger: slog.New(slog.DiscardHandler),
	})
	var writes atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-api-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet {
			writes.Add(1)
		}
		router.ServeHTTP(w, r.WithContext(authn.WithPrincipal(r.Context(), &authn.Principal{ID: 1})))
	}))
	t.Cleanup(server.Close)
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	connection, err := json.Marshal(map[string]string{"url": server.URL, "api_key": "synthetic-api-key", "ca_file": caPath})
	if err != nil {
		t.Fatal(err)
	}
	request := plugin.ReconcileRequest{
		Identity: plugin.Identity{Project: "adapter-test", Software: "AdapterApp", Destination: "woodstar"},
		Config:   connection,
		Metadata: json.RawMessage(fmt.Sprintf(`{"targets":{"include":[{"label_id":%d,"package":{"strategy":"latest"},"actions":["managed_installs"]}],"exclude":[{"label_id":%d}]}}`, included.ID, excluded.ID)),
	}
	setPkginfo(t, &request, fmt.Sprintf(`{
		"name":"AdapterApp", "version":"1.0", "installer_type":"nopkg",
		"description":"  Managed description  ","category":"Keep category","developer":"Keep publisher",
		"minimum_os_version":"  14.0  ","notes":"Keep notes","unattended_install":true,"supported_architectures":["arm64"],
		"force_install_after_date":"2026-10-01T10:00:00.123456789+10:00",
		"preinstall_alert":{"alert_title":"Keep title","alert_detail":"Original detail","ok_label":"Continue"},
		"requires":["%s--%s"],"update_for":["%s--%s"]
	}`, dependency.Name, dependencyPackage.Version, dependency.Name, dependencyPackage.Version))
	call := func(method string) plugin.ReconcileResponse {
		t.Helper()
		request.Method = method
		response, err := Handle(ctx, request)
		if err != nil {
			readback := request
			readback.Method = "plan"
			remaining, planErr := Handle(ctx, readback)
			for _, change := range remaining.Changes {
				t.Logf("remaining %s: before=%s after=%s", change.Field, change.Before, change.After)
			}
			if planErr != nil {
				t.Logf("readback plan: %v", planErr)
			}
			t.Fatalf("%s metadata %s: %v", method, request.Metadata, err)
		}
		return response
	}
	assertCounts := func(want int) {
		t.Helper()
		var titles, versions int
		if err := db.QueryRow(ctx, `SELECT (SELECT count(*) FROM munki_software), (SELECT count(*) FROM munki_packages)`).Scan(&titles, &versions); err != nil {
			t.Fatal(err)
		}
		if titles != want || versions != want {
			t.Fatalf("software/package counts = %d/%d, want %d/%d", titles, versions, want, want)
		}
	}
	if response := call("plan"); len(response.Changes) == 0 || writes.Load() != 0 {
		t.Fatalf("initial plan: changes=%+v writes=%d", response.Changes, writes.Load())
	}
	assertCounts(1)
	first := call("apply")
	request.Binding = first.Binding
	var identity struct {
		SoftwareID int64 `json:"software_id"`
		PackageID  int64 `json:"package_id"`
	}
	if err := json.Unmarshal(first.Binding, &identity); err != nil || identity.SoftwareID == 0 || identity.PackageID == 0 {
		t.Fatalf("created binding = %s, error=%v", first.Binding, err)
	}
	read := func() (*software.Software, *packages.Package, software.Targets) {
		t.Helper()
		title, err := softwareStore.GetByID(ctx, identity.SoftwareID)
		if err != nil {
			t.Fatal(err)
		}
		pkg, err := packageStore.GetByID(ctx, identity.PackageID)
		if err != nil {
			t.Fatal(err)
		}
		targets, err := softwareStore.TargetsForSoftware(ctx, identity.SoftwareID)
		if err != nil {
			t.Fatal(err)
		}
		return title, pkg, targets
	}
	assertConverged := func() {
		t.Helper()
		before := writes.Load()
		for _, method := range []string{"plan", "apply"} {
			response := call(method)
			if len(response.Changes) != 0 || writes.Load() != before {
				t.Fatalf("unchanged %s: changes=%+v writes=%d, want %d", method, response.Changes, writes.Load(), before)
			}
			var recovered struct {
				SoftwareID int64 `json:"software_id"`
				PackageID  int64 `json:"package_id"`
			}
			if err := json.Unmarshal(response.Binding, &recovered); err != nil || recovered != identity {
				t.Fatalf("identity changed: binding=%s error=%v", response.Binding, err)
			}
		}
		assertCounts(2)
	}
	title, pkg, targets := read()
	wantDeadline := time.Date(2026, time.October, 1, 0, 0, 0, 123456000, time.UTC)
	if title.Name != "AdapterApp" || title.Description != "Managed description" || pkg.Version != "1.0" || pkg.MinimumOSVersion != "14.0" || pkg.InstallerObjectID != nil || pkg.ForceInstallAfterDate == nil || !pkg.ForceInstallAfterDate.Equal(wantDeadline) {
		t.Fatalf("created defaults and normalization: software=%+v package=%+v", title, pkg)
	}
	wantReferences := []packages.PackageReference{{SoftwareID: dependency.ID, SoftwareName: dependency.Name, PackageID: dependencyPackage.ID, PackageVersion: dependencyPackage.Version}}
	if !reflect.DeepEqual(pkg.Requires, wantReferences) || !reflect.DeepEqual(pkg.UpdateFor, wantReferences) {
		t.Fatalf("native references changed: requires=%+v update_for=%+v", pkg.Requires, pkg.UpdateFor)
	}
	if len(targets.Include) != 1 || targets.Include[0].LabelID != included.ID || len(targets.Exclude) != 1 || targets.Exclude[0].LabelID != excluded.ID {
		t.Fatalf("created targets = %+v", targets)
	}
	assertConverged()

	request.Metadata = json.RawMessage(`{"targets":{"include":[]}}`)
	setPkginfo(t, &request, `{"name":"AdapterApp","version":"1.0","installer_type":"nopkg","preinstall_alert":{"alert_detail":"Updated detail"}}`)
	before := writes.Load()
	if response := call("plan"); len(response.Changes) == 0 || writes.Load() != before {
		t.Fatalf("update plan: changes=%+v writes=%d, want %d", response.Changes, writes.Load(), before)
	}
	call("apply")
	title, pkg, targets = read()
	if title.Description != "Managed description" || title.Category != "Keep category" || title.Developer != "Keep publisher" || len(targets.Include) != 0 || len(targets.Exclude) != 1 || targets.Exclude[0].LabelID != excluded.ID {
		t.Fatalf("omitted software fields lost: software=%+v targets=%+v", title, targets)
	}
	wantAlert := packages.PackageAlert{Enabled: true, Title: "Keep title", Detail: "Updated detail", OKLabel: "Continue"}
	if pkg.PreinstallAlert != wantAlert || pkg.Notes != "Keep notes" || !pkg.UnattendedInstall || pkg.MinimumOSVersion != "14.0" || pkg.ForceInstallAfterDate == nil || !pkg.ForceInstallAfterDate.Equal(wantDeadline) || !reflect.DeepEqual(pkg.Requires, wantReferences) || !reflect.DeepEqual(pkg.UpdateFor, wantReferences) {
		t.Fatalf("omitted package fields lost: %+v", pkg)
	}
	assertConverged()

	request.Metadata = json.RawMessage(`{"targets":{"exclude":[]}}`)
	setPkginfo(t, &request, `{
		"name":"AdapterApp","version":"1.0","installer_type":"nopkg",
		"description":null,"category":"","unattended_install":false,"notes":null,
		"minimum_os_version":"","force_install_after_date":null,
		"requires":[],"update_for":[],"supported_architectures":[],"preinstall_alert":null
	}`)
	call("apply")
	title, pkg, targets = read()
	if title.Description != "" || title.Category != "" || title.Developer != "Keep publisher" || len(targets.Include)+len(targets.Exclude) != 0 {
		t.Fatalf("software clearing changed ownership: software=%+v targets=%+v", title, targets)
	}
	wantAlert.Enabled = false
	if pkg.UnattendedInstall || pkg.Notes != "" || pkg.MinimumOSVersion != "" || pkg.ForceInstallAfterDate != nil || len(pkg.Requires)+len(pkg.UpdateFor)+len(pkg.SupportedArchitectures) != 0 || pkg.PreinstallAlert != wantAlert {
		t.Fatalf("package clearing changed ownership: %+v", pkg)
	}
	assertConverged()
	request.Binding = nil
	before = writes.Load()
	request.Method = "apply"
	if _, err := Handle(ctx, request); err == nil || writes.Load() != before {
		t.Fatalf("lost binding error=%v writes=%d", err, writes.Load())
	}
}

type testAuthorizer struct{}

func (testAuthorizer) CanAll(context.Context, int64, ...authz.Requirement) (bool, error) {
	return true, nil
}
