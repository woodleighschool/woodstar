//go:build postgres

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestMunkiSoftwareDeploymentAPI(t *testing.T) { //nolint:funlen,cyclop // One HTTP contract lifecycle shares one isolated database and router.
	db, ctx := testdb.Open(t)
	packageStore := packages.NewStore(db, nil)
	softwareStore := munkisoftware.NewStore(db, nil, packageStore)
	packageService := munki.NewPackageService(munki.PackageServiceDependencies{
		Packages:               packageStore,
		DesiredPackagesChanged: func() {},
	})
	hostStateStore := munki.NewStore(db)

	var softwareID, packageID, labelID, hostID int64
	if err := db.Pool().QueryRow(ctx, `INSERT INTO munki_software (name) VALUES ('Deployment API') RETURNING id`).Scan(&softwareID); err != nil {
		t.Fatalf("insert software: %v", err)
	}
	if err := db.Pool().QueryRow(ctx, `INSERT INTO munki_packages (software_id, version, installer_type) VALUES ($1, '3.0', 'nopkg') RETURNING id`, softwareID).Scan(&packageID); err != nil {
		t.Fatalf("insert package: %v", err)
	}
	if err := db.Pool().QueryRow(ctx, `INSERT INTO labels (name, label_type, label_membership_type) VALUES ('Deployment API Hosts', 'regular', 'manual') RETURNING id`).Scan(&labelID); err != nil {
		t.Fatalf("insert label: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
INSERT INTO munki_software_targets (software_id, direction, position, label_id, actions, package_selection, pinned_package_id)
VALUES ($1, 'include', 0, $2, ARRAY['managed_installs']::munki_manifest_action[], 'specific', $3)`, softwareID, labelID, packageID); err != nil {
		t.Fatalf("insert target: %v", err)
	}
	if err := db.Pool().QueryRow(ctx, `
INSERT INTO hosts (hardware_uuid, display_name, hardware_serial)
VALUES ('deployment-api-host', 'Deployment API Host', 'API-SERIAL') RETURNING id`).Scan(&hostID); err != nil {
		t.Fatalf("insert host: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `INSERT INTO label_membership (label_id, host_id) VALUES ($1, $2)`, labelID, hostID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	router := hostTestRouter(t, func(api huma.API) {
		registerListMunkiSoftware(api, softwareStore, discardLogger())
		registerCreateMunkiSoftware(api, softwareStore, packageService, discardLogger())
		registerGetMunkiSoftware(api, softwareStore, packageService, discardLogger())
		registerPutMunkiSoftware(api, softwareStore, packageService, discardLogger())
		registerListMunkiSoftwareHosts(api, softwareStore, discardLogger())
		registerHostMunkiState(api, hostStateStore, discardLogger())
		registerHostMunkiSoftware(api, softwareStore, discardLogger())
	})

	list := hostAPIRequest(t, router, http.MethodGet, "/api/munki/software", "")
	assertStatus(t, list, http.StatusOK, "list deployment software")
	var softwarePage Page[munkisoftware.SoftwareWithDeployment]
	decodeJSON(t, list, &softwarePage)
	if len(softwarePage.Items) != 1 ||
		softwarePage.Items[0].Deployment.AssignedCount != 1 ||
		softwarePage.Items[0].Deployment.InstalledCount != 0 {
		t.Fatalf("software page = %+v, want one unobserved assignment", softwarePage)
	}

	detail := hostAPIRequest(t, router, http.MethodGet, fmt.Sprintf("/api/munki/software/%d", softwareID), "")
	assertStatus(t, detail, http.StatusOK, "get deployment software")
	var detailBody munkiSoftwareDetail
	decodeJSON(t, detail, &detailBody)
	if detailBody.Deployment.AssignedCount != 1 || len(detailBody.Deployment.Packages) != 1 {
		t.Fatalf("detail deployment = %+v, want one package assignment", detailBody.Deployment)
	}

	hostsPath := fmt.Sprintf("/api/munki/software/%d/hosts", softwareID)
	hosts := hostAPIRequest(t, router, http.MethodGet, hostsPath, "")
	assertStatus(t, hosts, http.StatusOK, "list assigned hosts")
	var hostPage Page[munkisoftware.DeploymentHost]
	decodeJSON(t, hosts, &hostPage)
	if hostPage.Count != 1 || len(hostPage.Items) != 1 ||
		hostPage.Items[0].ReportState != munkisoftware.ReportNotContacted {
		t.Fatalf("assigned hosts = %+v, want one not-contacted host", hostPage)
	}
	filteredHosts := hostAPIRequest(t, router, http.MethodGet, hostsPath+"?action=managed_installs", "")
	assertStatus(t, filteredHosts, http.StatusOK, "filter assigned hosts")
	decodeJSON(t, filteredHosts, &hostPage)
	if hostPage.Count != 1 || len(hostPage.Items) != 1 {
		t.Fatalf("filtered assigned hosts = %+v, want one host", hostPage)
	}
	filteredHosts = hostAPIRequest(t, router, http.MethodGet, hostsPath+"?status=not_installed", "")
	assertStatus(t, filteredHosts, http.StatusOK, "filter unreported deployment status")
	decodeJSON(t, filteredHosts, &hostPage)
	if hostPage.Count != 0 || len(hostPage.Items) != 0 {
		t.Fatalf("unreported deployment status hosts = %+v, want none", hostPage)
	}
	assertStatus(
		t,
		hostAPIRequest(t, router, http.MethodGet, hostsPath+"?sort=bad", ""),
		http.StatusUnprocessableEntity,
		"reject invalid assigned-host sort",
	)

	created := hostAPIRequest(t, router, http.MethodPost, "/api/munki/software", `{"name":"Created Deployment","targets":{"include":[],"exclude":[]}}`)
	assertStatus(t, created, http.StatusCreated, "create deployment software")
	var createdBody munkiSoftwareDetail
	decodeJSON(t, created, &createdBody)
	if createdBody.Deployment.Packages == nil {
		t.Fatalf("created deployment uses nil collections: %+v", createdBody.Deployment)
	}
	updated := hostAPIRequest(t, router, http.MethodPut, fmt.Sprintf("/api/munki/software/%d", createdBody.ID), `{"description":"updated","targets":{"include":[],"exclude":[]}}`)
	assertStatus(t, updated, http.StatusOK, "update deployment software")
	var updatedBody munkiSoftwareDetail
	decodeJSON(t, updated, &updatedBody)
	if updatedBody.Deployment.Packages == nil {
		t.Fatalf("updated deployment uses nil collections: %+v", updatedBody.Deployment)
	}

	hostStatePath := fmt.Sprintf("/api/hosts/%d/munki", hostID)
	assertStatus(t, hostAPIRequest(t, router, http.MethodGet, hostStatePath, ""), http.StatusNotFound, "host before Munki report")
	if _, err := db.Pool().Exec(ctx, `INSERT INTO host_heartbeats (host_id, source) VALUES ($1, 'munki')`, hostID); err != nil {
		t.Fatalf("record Munki contact: %v", err)
	}
	assertStatus(t, hostAPIRequest(t, router, http.MethodGet, hostStatePath, ""), http.StatusNotFound, "host after contact before report")
	attemptedAt := time.Date(2026, 8, 2, 14, 0, 0, 0, time.UTC)
	if err := hostStateStore.ApplyEnvelope(ctx, munki.EnvelopeResult{
		HostID:      hostID,
		AttemptedAt: attemptedAt,
		Complete:    true,
		HasReport:   true,
		Items: []munki.ItemObservation{{
			Name:             "Deployment API",
			Installed:        true,
			InstalledVersion: " 3.0 ",
		}},
	}); err != nil {
		t.Fatalf("apply envelope: %v", err)
	}
	detail = hostAPIRequest(t, router, http.MethodGet, fmt.Sprintf("/api/munki/software/%d", softwareID), "")
	assertStatus(t, detail, http.StatusOK, "get software after Munki report")
	decodeJSON(t, detail, &detailBody)
	if detailBody.Deployment.AssignedCount != 1 ||
		detailBody.Deployment.ReportingCount != 1 ||
		detailBody.Deployment.InstalledCount != 1 ||
		len(detailBody.Deployment.Packages) != 1 ||
		detailBody.Deployment.Packages[0].Version != "3.0" {
		t.Fatalf("software deployment = %+v, want one installed package assignment", detailBody.Deployment)
	}
	hosts = hostAPIRequest(t, router, http.MethodGet, hostsPath, "")
	assertStatus(t, hosts, http.StatusOK, "list assigned hosts after Munki report")
	decodeJSON(t, hosts, &hostPage)
	if hostPage.Count != 1 || !hostPage.Items[0].Installed ||
		hostPage.Items[0].InstalledVersion != "3.0" ||
		hostPage.Items[0].ReportState != munkisoftware.ReportCurrent ||
		hostPage.Items[0].Status == nil ||
		*hostPage.Items[0].Status != munkisoftware.StatusUpToDate {
		t.Fatalf("assigned host = %+v, want current installed 3.0", hostPage)
	}
	filteredHosts = hostAPIRequest(t, router, http.MethodGet, hostsPath+"?status=up_to_date", "")
	assertStatus(t, filteredHosts, http.StatusOK, "filter up-to-date hosts")
	decodeJSON(t, filteredHosts, &hostPage)
	if hostPage.Count != 1 || len(hostPage.Items) != 1 || !hostPage.Items[0].Installed {
		t.Fatalf("up-to-date hosts = %+v, want one installed host", hostPage)
	}
	hostSoftware := hostAPIRequest(t, router, http.MethodGet, fmt.Sprintf("/api/hosts/%d/munki/software", hostID), "")
	assertStatus(t, hostSoftware, http.StatusOK, "list host Munki software status")
	var desired Page[munkisoftware.HostManifestSoftware]
	decodeJSON(t, hostSoftware, &desired)
	if desired.Count != 1 ||
		desired.Items[0].Status == nil ||
		*desired.Items[0].Status != munkisoftware.StatusUpToDate {
		body, _ := json.Marshal(desired)
		t.Fatalf("host desired software = %s, want up to date", body)
	}
}
