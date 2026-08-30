//go:build e2e

package e2e

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/storage/capability"
	"github.com/woodleighschool/woodstar/test/e2e/adminapi"
)

const (
	tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
	munkiSecret   = "munki-integration-secret-0123456789abcdef"
)

func TestMunki(t *testing.T) { //nolint:cyclop,funlen,gocognit // Linear product lifecycle; splitting would hide the order being proved.
	const (
		serial             = "C02WOODSTARMUNKI"
		secondSerial       = "C02WOODSTARMUNKI2"
		softwareName       = "WoodstarIntegrationApp"
		secondSoftwareName = "WoodstarSecondIntegrationApp"
	)

	server := startTestServer(t)
	server.redact(munkiSecret)
	transferClient := verifyingClient(t, server.CACertificate)

	provisionAdmin(
		t,
		server,
		"admin@woodstar.test",
		"Integration Administrator",
		"integration-admin-password",
	)
	baseURL, err := url.Parse(server.BaseURL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	cookies := server.Client.Jar.Cookies(baseURL)
	if len(cookies) == 0 {
		t.Fatal("admin client did not retain the login session cookie")
	}

	database, err := pgx.Connect(t.Context(), server.DatabaseURL)
	if err != nil {
		t.Fatalf("connect to isolated database: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), databaseOperationTimeout)
		defer cancel()
		if closeErr := database.Close(ctx); closeErr != nil {
			t.Errorf("close isolated database connection: %v", closeErr)
		}
	})
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "woodstar_session" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("admin client did not retain woodstar_session")
	}
	var storedSessionToken string
	if err := database.QueryRow(t.Context(), "SELECT token FROM sessions").Scan(&storedSessionToken); err != nil {
		t.Fatalf("query stored session token: %v", err)
	}
	tokenHash := sha256.Sum256([]byte(sessionCookie.Value))
	wantStoredToken := base64.RawURLEncoding.EncodeToString(tokenHash[:])
	if storedSessionToken != wantStoredToken {
		t.Fatal("session store retained the bearer token instead of its SHA-256 hash")
	}

	var hostID int64
	err = database.QueryRow(
		t.Context(),
		`INSERT INTO hosts (hardware_uuid, hardware_serial, os_platform)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		"50E9D0D5-499F-4E43-BB52-55A30F7986E1",
		serial,
		"darwin",
	).Scan(&hostID)
	if err != nil {
		t.Fatalf("seed canonical macOS host: %v", err)
	}
	var allHostsLabelID int64
	if err := database.QueryRow(
		t.Context(),
		"SELECT id FROM labels WHERE builtin_key = 'all-hosts'",
	).Scan(&allHostsLabelID); err != nil {
		t.Fatalf("load migration-seeded all-hosts label: %v", err)
	}
	if _, err := database.Exec(
		t.Context(),
		"INSERT INTO label_membership (label_id, host_id) VALUES ($1, $2)",
		allHostsLabelID,
		hostID,
	); err != nil {
		t.Fatalf("seed all-hosts membership: %v", err)
	}

	createdSecret := createAgentSecret(t, server, adminapi.AgentSecretCreateAgentMunki, munkiSecret)
	if createdSecret.Id <= 0 || createdSecret.Agent != "munki" {
		t.Fatalf("created agent secret = %+v, want active Munki secret", createdSecret)
	}

	installerBytes := bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, 200)
	installerSum := sha256.Sum256(installerBytes)
	installerSHA256 := hex.EncodeToString(installerSum[:])
	capabilityIssuedAfter := time.Now()
	createdInstaller, err := server.Admin.CreateMunkiPackageInstallerUploadWithResponse(
		t.Context(),
		adminapi.MunkiPackageInstallerUploadRequest{
			Filename:  "WoodstarIntegration.pkg",
			SizeBytes: int64(len(installerBytes)),
		},
	)
	capabilityIssuedBefore := time.Now()
	createdInstaller = requireAPIResponse(
		t,
		"create package installer",
		http.StatusCreated,
		createdInstaller,
		err,
	)
	installerTarget := createdInstaller.JSON201
	installerUploadAction := directPackageInstallerUpload(t, installerTarget)
	if installerTarget.ObjectId <= 0 || installerUploadAction.Method != http.MethodPut ||
		installerUploadAction.Strategy != "direct-put" {
		t.Fatalf(
			"installer upload target id/method/strategy = %d/%q/%q, want positive/PUT/direct-put",
			installerTarget.ObjectId,
			installerUploadAction.Method,
			installerUploadAction.Strategy,
		)
	}
	var secondHostID int64
	err = database.QueryRow(
		t.Context(),
		`INSERT INTO hosts (hardware_uuid, hardware_serial, os_platform)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		"9FB5D1F5-4DDD-4B3B-839A-8C05E904F70A",
		secondSerial,
		"darwin",
	).Scan(&secondHostID)
	if err != nil {
		t.Fatalf("seed second canonical macOS host: %v", err)
	}
	secondLabelResponse, err := server.Admin.CreateLabelWithResponse(
		t.Context(),
		adminapi.LabelMutation{
			Name:                "Munki Second Integration Host",
			LabelMembershipType: new(adminapi.LabelMutationLabelMembershipType("manual")),
			HostIds:             new([]int64{secondHostID}),
		},
	)
	secondLabelResponse = requireAPIResponse(
		t,
		"create second-host Munki label",
		http.StatusCreated,
		secondLabelResponse,
		err,
	)
	if secondLabelResponse.JSON201 == nil || secondLabelResponse.JSON201.Id <= 0 {
		t.Fatal("create second-host Munki label returned no label")
	}
	secondHostLabelID := secondLabelResponse.JSON201.Id
	assertStorageCapabilityTTL(
		t,
		installerUploadAction.Url,
		server.StorageCapabilityKey,
		capability.OpPut,
		capabilityIssuedAfter,
		capabilityIssuedBefore,
	)
	installerUpload, err := http.NewRequestWithContext(
		t.Context(),
		string(installerUploadAction.Method),
		installerUploadAction.Url,
		bytes.NewReader(installerBytes),
	)
	if err != nil {
		t.Fatal("create installer upload capability request")
	}
	if installerUploadAction.Headers != nil {
		for name, value := range *installerUploadAction.Headers {
			installerUpload.Header.Set(name, value)
		}
	}
	installerUploadResponse, err := transferClient.Do(installerUpload)
	if err != nil {
		t.Fatal("upload installer through returned capability")
	}
	drainAndClose(t, installerUploadResponse)
	if installerUploadResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("installer upload status = %d, want %d", installerUploadResponse.StatusCode, http.StatusNoContent)
	}

	finalizedInstaller, err := server.Admin.CompleteMunkiPackageInstallerUploadWithResponse(
		t.Context(),
		installerTarget.ObjectId,
	)
	finalizedInstaller = requireAPIResponse(
		t,
		"finalize package installer",
		http.StatusOK,
		finalizedInstaller,
		err,
	)
	if finalizedInstaller.JSON200 == nil {
		t.Fatal("finalize package installer returned no JSON body")
	}
	installer := *finalizedInstaller.JSON200
	if installer.Id != installerTarget.ObjectId || installer.Filename != "WoodstarIntegration.pkg" ||
		installer.ContentType != "application/octet-stream" || installer.SizeBytes == nil ||
		*installer.SizeBytes != int64(len(installerBytes)) || installer.Sha256 == nil ||
		*installer.Sha256 != installerSHA256 ||
		installer.ContentUrl != "/api/munki/package-installers/"+
			strconv.FormatInt(installer.Id, 10)+"/content" {
		t.Fatal("finalized installer did not contain the expected server-derived metadata")
	}
	installerContentResponse := getResponse(t, server.AdminHTTP, server.BaseURL+installer.ContentUrl)
	if got := readAndClose(t, installerContentResponse); installerContentResponse.StatusCode != http.StatusOK ||
		!bytes.Equal(got, installerBytes) {
		t.Fatalf(
			"admin installer content status/body = %d/%d, want 200/exact",
			installerContentResponse.StatusCode,
			len(got),
		)
	}
	agentOnAdminRequest, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		server.BaseURL+installer.ContentUrl,
		nil,
	)
	if err != nil {
		t.Fatal("create agent request for admin content route")
	}
	agentOnAdminRequest.Header.Set("Authorization", "Bearer "+munkiSecret)
	agentOnAdminResponse, err := transferClient.Do(agentOnAdminRequest)
	if err != nil {
		t.Fatalf("request admin content with agent secret: %v", err)
	}
	drainAndClose(t, agentOnAdminResponse)
	if agentOnAdminResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("agent secret on admin content status = %d, want 401", agentOnAdminResponse.StatusCode)
	}

	targets := adminapi.MunkiTargets{
		Include: []adminapi.MunkiInclude{{
			LabelId: allHostsLabelID,
			Package: adminapi.MunkiPackageSelector{Strategy: "latest"},
			Actions: []adminapi.MunkiIncludeActions{"managed_installs"},
		}},
		Exclude: []adminapi.LabelRef{},
	}
	createdSoftware, err := server.Admin.CreateMunkiSoftwareWithResponse(
		t.Context(),
		adminapi.MunkiCreateMutation{
			Name:        softwareName,
			DisplayName: new("Woodstar Integration App"),
			Description: new("Compiled Munki repository lifecycle fixture."),
			Category:    new("Testing"),
			Developer:   new("Example Developer"),
			Targets:     targets,
		},
	)
	createdSoftware = requireAPIResponse(t, "create software", http.StatusCreated, createdSoftware, err)
	if createdSoftware.JSON201 == nil || createdSoftware.JSON201.Id <= 0 ||
		createdSoftware.JSON201.Name != softwareName {
		t.Fatalf("created software = %+v, want %s", createdSoftware.JSON201, softwareName)
	}
	software := *createdSoftware.JSON201
	firstIconBytes, err := base64.StdEncoding.DecodeString(tinyPNGBase64)
	if err != nil {
		t.Fatalf("decode first icon fixture: %v", err)
	}
	firstIcon := attachMunkiIcon(t, server, transferClient, software.Id, "WoodstarIntegrationApp.png", firstIconBytes)
	firstIconName := strconv.FormatInt(firstIcon.Id, 10) + "-" + firstIcon.Filename

	secondInstallerBytes := bytes.Repeat([]byte{0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00}, 200)
	createdSecondInstaller, err := server.Admin.CreateMunkiPackageInstallerUploadWithResponse(
		t.Context(),
		adminapi.MunkiPackageInstallerUploadRequest{
			Filename:  "WoodstarSecondIntegration.pkg",
			SizeBytes: int64(len(secondInstallerBytes)),
		},
	)
	createdSecondInstaller = requireAPIResponse(
		t,
		"create second package installer",
		http.StatusCreated,
		createdSecondInstaller,
		err,
	)
	secondInstallerTarget := createdSecondInstaller.JSON201
	secondInstallerUpload := directPackageInstallerUpload(t, secondInstallerTarget)
	secondInstallerRequest, err := http.NewRequestWithContext(
		t.Context(),
		string(secondInstallerUpload.Method),
		secondInstallerUpload.Url,
		bytes.NewReader(secondInstallerBytes),
	)
	if err != nil {
		t.Fatal("create second installer upload capability request")
	}
	if secondInstallerUpload.Headers != nil {
		for name, value := range *secondInstallerUpload.Headers {
			secondInstallerRequest.Header.Set(name, value)
		}
	}
	secondInstallerUploadResponse, err := transferClient.Do(secondInstallerRequest)
	if err != nil {
		t.Fatal("upload second installer through returned capability")
	}
	drainAndClose(t, secondInstallerUploadResponse)
	if secondInstallerUploadResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("second installer upload status = %d, want %d", secondInstallerUploadResponse.StatusCode, http.StatusNoContent)
	}
	finalizedSecondInstaller, err := server.Admin.CompleteMunkiPackageInstallerUploadWithResponse(
		t.Context(),
		secondInstallerTarget.ObjectId,
	)
	finalizedSecondInstaller = requireAPIResponse(
		t,
		"finalize second package installer",
		http.StatusOK,
		finalizedSecondInstaller,
		err,
	)
	if finalizedSecondInstaller.JSON200 == nil {
		t.Fatal("finalize second package installer returned no JSON body")
	}
	secondInstaller := *finalizedSecondInstaller.JSON200
	createdSecondSoftware, err := server.Admin.CreateMunkiSoftwareWithResponse(
		t.Context(),
		adminapi.MunkiCreateMutation{
			Name:        secondSoftwareName,
			DisplayName: new("Woodstar Second Integration App"),
			Targets: adminapi.MunkiTargets{
				Include: []adminapi.MunkiInclude{{
					LabelId: secondHostLabelID,
					Package: adminapi.MunkiPackageSelector{Strategy: "latest"},
					Actions: []adminapi.MunkiIncludeActions{"managed_installs"},
				}},
				Exclude: []adminapi.LabelRef{},
			},
		},
	)
	createdSecondSoftware = requireAPIResponse(
		t,
		"create second-host software",
		http.StatusCreated,
		createdSecondSoftware,
		err,
	)
	if createdSecondSoftware.JSON201 == nil || createdSecondSoftware.JSON201.Id <= 0 {
		t.Fatal("create second-host software returned no software")
	}
	secondSoftware := *createdSecondSoftware.JSON201
	secondIconBytes := append([]byte(nil), firstIconBytes...)
	secondIcon := attachMunkiIcon(
		t,
		server,
		transferClient,
		secondSoftware.Id,
		"WoodstarSecondIntegrationApp.png",
		secondIconBytes,
	)
	secondIconName := strconv.FormatInt(secondIcon.Id, 10) + "-" + secondIcon.Filename

	createdPackage, err := server.Admin.CreateMunkiPackageWithResponse(
		t.Context(),
		adminapi.MunkiPackageCreateMutation{
			SoftwareId:        software.Id,
			Version:           "1.0",
			InstallerType:     new(adminapi.MunkiPackageCreateMutationInstallerType("pkg")),
			InstallerObjectId: new(installer.Id),
		},
	)
	createdPackage = requireAPIResponse(t, "create package", http.StatusCreated, createdPackage, err)
	if createdPackage.JSON201 == nil {
		t.Fatal("create package returned no JSON body")
	}
	pkg := *createdPackage.JSON201
	installerItemLocation := fmt.Sprintf(
		"packages/%d/installer/%s",
		pkg.Id,
		installer.Filename,
	)
	if pkg.Software.Id != software.Id || pkg.Version != "1.0" || pkg.InstallerType != "pkg" ||
		pkg.InstallerObjectId == nil || *pkg.InstallerObjectId != installer.Id ||
		pkg.InstallerFile == nil || pkg.InstallerFile.Filename != installer.Filename ||
		pkg.InstallerFile.InstallerItemLocation != installerItemLocation ||
		pkg.InstallerFile.SizeBytes != int64(len(installerBytes)) ||
		pkg.InstallerFile.Sha256 != installerSHA256 {
		t.Fatalf("created package = %+v, want finalized installer version", pkg)
	}
	initialResources, err := server.Admin.ListMunkiClientResourcesWithResponse(t.Context(), nil)
	initialResources = requireAPIResponse(
		t,
		"list undeployed client resources",
		http.StatusOK,
		initialResources,
		err,
	)
	if initialResources.JSON200 == nil || initialResources.JSON200.Count != 0 ||
		len(initialResources.JSON200.Items) != 0 {
		t.Fatalf("initial client resources = %+v, want empty page", initialResources.JSON200)
	}

	bannerBytes, err := base64.StdEncoding.DecodeString(tinyPNGBase64)
	if err != nil {
		t.Fatalf("decode tiny PNG fixture: %v", err)
	}
	bannerConfig, err := png.DecodeConfig(bytes.NewReader(bannerBytes))
	if err != nil {
		t.Fatalf("decode tiny PNG fixture dimensions: %v", err)
	}
	if bannerConfig.Width != 1 || bannerConfig.Height != 1 {
		t.Fatalf("tiny PNG dimensions = %dx%d, want 1x1", bannerConfig.Width, bannerConfig.Height)
	}
	createdBanner, err := server.Admin.CreateMunkiClientResourcesBannerUploadWithResponse(
		t.Context(),
		adminapi.MunkiDirectUploadRequest{
			Filename: "banner.png",
		},
	)
	createdBanner = requireAPIResponse(t, "create banner upload", http.StatusCreated, createdBanner, err)
	bannerTarget := createdBanner.JSON201
	if bannerTarget == nil {
		t.Fatal("create banner upload returned no JSON body")
	}
	bannerUploadAction := bannerTarget.Upload
	if bannerTarget.ObjectId <= 0 || bannerUploadAction.Method != http.MethodPut ||
		bannerUploadAction.Strategy != "direct-put" {
		t.Fatalf(
			"banner upload target id/method/strategy = %d/%q/%q, want positive/PUT/direct-put",
			bannerTarget.ObjectId,
			bannerUploadAction.Method,
			bannerUploadAction.Strategy,
		)
	}
	createdSecondPackage, err := server.Admin.CreateMunkiPackageWithResponse(
		t.Context(),
		adminapi.MunkiPackageCreateMutation{
			SoftwareId:        secondSoftware.Id,
			Version:           "2.0",
			InstallerType:     new(adminapi.MunkiPackageCreateMutationInstallerType("pkg")),
			InstallerObjectId: new(secondInstaller.Id),
		},
	)
	createdSecondPackage = requireAPIResponse(
		t,
		"create second-host package",
		http.StatusCreated,
		createdSecondPackage,
		err,
	)
	if createdSecondPackage.JSON201 == nil || createdSecondPackage.JSON201.InstallerFile == nil {
		t.Fatal("create second-host package returned no finalized installer")
	}
	secondPackage := *createdSecondPackage.JSON201
	secondInstallerItemLocation := secondPackage.InstallerFile.InstallerItemLocation
	bannerUpload, err := http.NewRequestWithContext(
		t.Context(),
		string(bannerUploadAction.Method),
		bannerUploadAction.Url,
		bytes.NewReader(bannerBytes),
	)
	if err != nil {
		t.Fatal("create banner upload capability request")
	}
	if bannerUploadAction.Headers != nil {
		for name, value := range *bannerUploadAction.Headers {
			bannerUpload.Header.Set(name, value)
		}
	}
	bannerUploadResponse, err := transferClient.Do(bannerUpload)
	if err != nil {
		t.Fatal("upload banner through returned capability")
	}
	drainAndClose(t, bannerUploadResponse)
	if bannerUploadResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("banner upload status = %d, want %d", bannerUploadResponse.StatusCode, http.StatusNoContent)
	}

	links := []adminapi.MunkiLink{{
		Label:         "Support",
		Target:        "https://support.woodstar.test/",
		OpenInBrowser: true,
	}}
	footerLinks := []adminapi.MunkiLink{{
		Label:  "Updates",
		Target: "munki://updates",
	}}
	builder := adminapi.MunkiBuilder{
		BannerObjectId: bannerTarget.ObjectId,
		BannerFit:      "cover",
		BannerFocalX:   50,
		Links:          links,
		FooterText:     "Managed by Example IT",
		FooterLinks:    footerLinks,
	}
	savedResources, err := server.Admin.CreateMunkiClientResourcesWithResponse(
		t.Context(),
		adminapi.MunkiClientResourcesMutation{
			Builder: &builder,
		},
	)
	savedResources = requireAPIResponse(t, "create client resources", http.StatusCreated, savedResources, err)
	if savedResources.JSON201 == nil {
		t.Fatal("create client resources returned no JSON body")
	}
	clientResources := *savedResources.JSON201
	bannerSum := sha256.Sum256(bannerBytes)
	bannerSHA256 := hex.EncodeToString(bannerSum[:])
	if clientResources.Id != 1 || clientResources.Custom || clientResources.Builder == nil {
		t.Fatal("saved client resources returned no builder state")
	}
	if clientResources.Builder.Banner.Id != bannerTarget.ObjectId ||
		clientResources.Builder.Banner.ContentType != "image/png" ||
		clientResources.Builder.Banner.SizeBytes == nil ||
		*clientResources.Builder.Banner.SizeBytes != int64(len(bannerBytes)) ||
		clientResources.Builder.Banner.Sha256 == nil ||
		*clientResources.Builder.Banner.Sha256 != bannerSHA256 ||
		clientResources.Builder.BannerFit != "cover" ||
		clientResources.Builder.BannerFocalX != 50 ||
		clientResources.Builder.FooterText != "Managed by Example IT" ||
		clientResources.Archive.ContentType != "application/zip" {
		t.Fatal("saved client resources did not contain the expected compiled banner state")
	}

	rereadSoftwareResponse, err := server.Admin.GetMunkiSoftwareWithResponse(t.Context(), software.Id)
	rereadSoftwareResponse = requireAPIResponse(
		t,
		"get software",
		http.StatusOK,
		rereadSoftwareResponse,
		err,
	)
	if rereadSoftwareResponse.JSON200 == nil {
		t.Fatal("get software returned no JSON body")
	}
	rereadSoftware := *rereadSoftwareResponse.JSON200
	if rereadSoftware.Id != software.Id || len(rereadSoftware.Packages) != 1 ||
		rereadSoftware.Packages[0].Id != pkg.Id || len(rereadSoftware.Targets.Include) != 1 ||
		rereadSoftware.Targets.Include[0].LabelId != allHostsLabelID ||
		rereadSoftware.Targets.Include[0].Package.Strategy != "latest" ||
		len(rereadSoftware.Targets.Include[0].Actions) != 1 ||
		rereadSoftware.Targets.Include[0].Actions[0] != "managed_installs" {
		t.Fatalf("re-read software = %+v, want saved package and all-hosts target", rereadSoftware)
	}
	listedResourcesResponse, err := server.Admin.ListMunkiClientResourcesWithResponse(t.Context(), nil)
	listedResourcesResponse = requireAPIResponse(
		t,
		"list client resources",
		http.StatusOK,
		listedResourcesResponse,
		err,
	)
	if listedResourcesResponse.JSON200 == nil || listedResourcesResponse.JSON200.Count != 1 ||
		len(listedResourcesResponse.JSON200.Items) != 1 ||
		listedResourcesResponse.JSON200.Items[0].Id != clientResources.Id {
		t.Fatalf("listed client resources = %+v, want only ID 1", listedResourcesResponse.JSON200)
	}
	rereadResourcesResponse, err := server.Admin.GetMunkiClientResourcesWithResponse(
		t.Context(),
		clientResources.Id,
	)
	rereadResourcesResponse = requireAPIResponse(
		t,
		"get client resources",
		http.StatusOK,
		rereadResourcesResponse,
		err,
	)
	if rereadResourcesResponse.JSON200 == nil {
		t.Fatal("get client resources returned no JSON body")
	}
	rereadClientResources := *rereadResourcesResponse.JSON200
	if rereadClientResources.Custom || rereadClientResources.Builder == nil {
		t.Fatal("re-read client resources returned no builder state")
	}
	if rereadClientResources.Builder.Banner.Id != bannerTarget.ObjectId ||
		rereadClientResources.Builder.Banner.ContentUrl != "/api/munki/client-resources/banner-uploads/"+
			strconv.FormatInt(bannerTarget.ObjectId, 10)+"/content" ||
		rereadClientResources.Builder.BannerFit != "cover" ||
		rereadClientResources.Builder.BannerFocalX != 50 ||
		len(rereadClientResources.Builder.Links) != 1 ||
		rereadClientResources.Builder.Links[0] != links[0] ||
		len(rereadClientResources.Builder.FooterLinks) != 1 ||
		rereadClientResources.Builder.FooterLinks[0] != footerLinks[0] {
		t.Fatal("re-read client resources did not match the saved public state")
	}
	bannerContentResponse := getResponse(
		t,
		server.AdminHTTP,
		server.BaseURL+rereadClientResources.Builder.Banner.ContentUrl,
	)
	if got := readAndClose(t, bannerContentResponse); bannerContentResponse.StatusCode != http.StatusOK ||
		!bytes.Equal(got, bannerBytes) {
		t.Fatalf("admin banner content status/body = %d/%d, want 200/exact", bannerContentResponse.StatusCode, len(got))
	}

	munkiClient := verifyingClient(t, server.CACertificate)
	munkiClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	manifestRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/manifests/"+serial,
		serial,
	)
	manifestStartedAt := time.Now()
	manifestResponse, err := munkiClient.Do(manifestRequest)
	manifestFinishedAt := time.Now()
	if err != nil {
		t.Fatalf("fetch Munki manifest: %v", err)
	}
	manifestBody := readAndClose(t, manifestResponse)
	if manifestResponse.StatusCode != http.StatusOK {
		t.Fatalf("manifest status = %d, want %d", manifestResponse.StatusCode, http.StatusOK)
	}
	if got := manifestResponse.Header.Get("Content-Type"); got != "application/x-plist" {
		t.Fatalf("manifest content type = %q, want application/x-plist", got)
	}
	manifestETag := manifestResponse.Header.Get("ETag")
	if manifestETag == "" {
		t.Fatal("manifest ETag is empty")
	}
	var manifest struct {
		Catalogs        []string `plist:"catalogs"`
		ManagedInstalls []string `plist:"managed_installs"`
	}
	if _, err := plist.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatalf("decode Munki manifest plist: %v", err)
	}
	if len(manifest.Catalogs) != 1 || manifest.Catalogs[0] != "woodstar" ||
		len(manifest.ManagedInstalls) != 1 || manifest.ManagedInstalls[0] != softwareName {
		t.Fatalf("manifest = %+v, want woodstar catalog and %s install", manifest, softwareName)
	}
	manifestHostResponse, err := server.Admin.GetHostWithResponse(t.Context(), hostID)
	manifestHostResponse = requireAPIResponse(
		t,
		"get host after Munki manifest",
		http.StatusOK,
		manifestHostResponse,
		err,
	)
	if manifestHostResponse.JSON200 == nil {
		t.Fatal("get host after Munki manifest returned no JSON body")
	}
	manifestHost := *manifestHostResponse.JSON200
	manifestHeartbeat := requireHeartbeat(t, manifestHost.Heartbeats, "munki")
	if len(manifestHost.Heartbeats) != 1 ||
		manifestHeartbeat.LastSeenAt.Before(manifestStartedAt.Add(-heartbeatTimeTolerance)) ||
		manifestHeartbeat.LastSeenAt.After(manifestFinishedAt.Add(heartbeatTimeTolerance)) ||
		manifestHost.LastContact == nil ||
		!manifestHost.LastContact.Equal(manifestHeartbeat.LastSeenAt) || manifestHost.Status != "offline" ||
		manifestHost.PublicIp != nil {
		t.Fatalf(
			"host after Munki manifest = %+v, heartbeat = %+v, want one bounded Munki contact without osquery state",
			manifestHost,
			manifestHeartbeat,
		)
	}
	secondManifestRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/manifests/"+secondSerial,
		secondSerial,
	)
	secondManifestRequest.Header.Set("If-None-Match", manifestETag)
	secondManifestResponse, err := munkiClient.Do(secondManifestRequest)
	if err != nil {
		t.Fatalf("fetch second-host Munki manifest with first-host ETag: %v", err)
	}
	secondManifestBody := readAndClose(t, secondManifestResponse)
	if secondManifestResponse.StatusCode != http.StatusOK || secondManifestResponse.Header.Get("ETag") == manifestETag {
		t.Fatalf(
			"second-host manifest status/etag = %d/%q, want 200 and an ETag distinct from %q",
			secondManifestResponse.StatusCode,
			secondManifestResponse.Header.Get("ETag"),
			manifestETag,
		)
	}
	var secondManifest struct {
		Catalogs        []string `plist:"catalogs"`
		ManagedInstalls []string `plist:"managed_installs"`
	}
	if _, err := plist.Unmarshal(secondManifestBody, &secondManifest); err != nil {
		t.Fatalf("decode second-host Munki manifest plist: %v", err)
	}
	if len(secondManifest.Catalogs) != 1 || secondManifest.Catalogs[0] != "woodstar" ||
		len(secondManifest.ManagedInstalls) != 1 || secondManifest.ManagedInstalls[0] != secondSoftwareName {
		t.Fatalf("second-host manifest = %+v, want woodstar catalog and %s install", secondManifest, secondSoftwareName)
	}
	missingSerialRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/manifests/"+serial,
		serial,
	)
	missingSerialRequest.Header.Del("X-Woodstar-Serial-Number")
	missingSerialResponse, err := munkiClient.Do(missingSerialRequest)
	if err != nil {
		t.Fatalf("fetch Munki manifest without serial: %v", err)
	}
	drainAndClose(t, missingSerialResponse)
	if missingSerialResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("manifest without serial status = %d, want 404", missingSerialResponse.StatusCode)
	}
	unknownSerialRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/manifests/"+serial,
		"C02WOODSTARUNKNOWN",
	)
	unknownSerialResponse, err := munkiClient.Do(unknownSerialRequest)
	if err != nil {
		t.Fatalf("fetch Munki manifest with unknown serial: %v", err)
	}
	drainAndClose(t, unknownSerialResponse)
	if unknownSerialResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("manifest with unknown serial status = %d, want 404", unknownSerialResponse.StatusCode)
	}

	cachedManifestRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/manifests/"+serial,
		serial,
	)
	cachedManifestRequest.Header.Set("If-None-Match", manifestETag)
	cachedManifestResponse, err := munkiClient.Do(cachedManifestRequest)
	if err != nil {
		t.Fatalf("fetch cached Munki manifest: %v", err)
	}
	cachedManifestBody := readAndClose(t, cachedManifestResponse)
	if cachedManifestResponse.StatusCode != http.StatusNotModified ||
		cachedManifestResponse.Header.Get("ETag") != manifestETag || len(cachedManifestBody) != 0 {
		t.Fatalf(
			"cached manifest status/etag/body = %d/%q/%d, want 304/%q/0",
			cachedManifestResponse.StatusCode,
			cachedManifestResponse.Header.Get("ETag"),
			len(cachedManifestBody),
			manifestETag,
		)
	}

	catalogRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/catalogs/woodstar",
		serial,
	)
	catalogResponse, err := munkiClient.Do(catalogRequest)
	if err != nil {
		t.Fatalf("fetch Munki catalog: %v", err)
	}
	catalogBody := readAndClose(t, catalogResponse)
	if catalogResponse.StatusCode != http.StatusOK {
		t.Fatalf("catalog status = %d, want %d", catalogResponse.StatusCode, http.StatusOK)
	}
	var catalog []struct {
		Name                  string `plist:"name"`
		Version               string `plist:"version"`
		InstallerItemLocation string `plist:"installer_item_location"`
		InstallerItemHash     string `plist:"installer_item_hash"`
		InstallerItemSize     int64  `plist:"installer_item_size"`
		IconName              string `plist:"icon_name"`
	}
	if _, err := plist.Unmarshal(catalogBody, &catalog); err != nil {
		t.Fatalf("decode Munki catalog plist: %v", err)
	}
	wantInstallerKiB := (int64(len(installerBytes)) + 1023) / 1024
	if len(catalog) != 1 || catalog[0].Name != softwareName || catalog[0].Version != "1.0" ||
		catalog[0].InstallerItemLocation != installerItemLocation ||
		catalog[0].InstallerItemHash != installerSHA256 ||
		catalog[0].InstallerItemSize != wantInstallerKiB || catalog[0].IconName != firstIconName {
		t.Fatalf("catalog = %+v, want package %s at %s", catalog, softwareName, installerItemLocation)
	}

	packageRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/pkgs/"+installerItemLocation,
		serial,
	)
	sessionOnlyPackageRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/pkgs/"+installerItemLocation,
		serial,
	)
	sessionOnlyPackageRequest.Header.Del("Authorization")
	sessionOnlyPackageRequest.Header.Del("X-Woodstar-Serial-Number")
	sessionOnlyPackageResponse, err := server.Client.Do(sessionOnlyPackageRequest)
	if err != nil {
		t.Fatalf("request agent package with admin session: %v", err)
	}
	drainAndClose(t, sessionOnlyPackageResponse)
	if sessionOnlyPackageResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("admin session on agent package status = %d, want 401", sessionOnlyPackageResponse.StatusCode)
	}
	packageResponse, err := munkiClient.Do(packageRequest)
	if err != nil {
		t.Fatalf("fetch package: %v", err)
	}
	deliveredInstaller := readAndClose(t, packageResponse)
	if packageResponse.StatusCode != http.StatusOK || !bytes.Equal(deliveredInstaller, installerBytes) ||
		packageResponse.Header.Get("Content-Type") != installer.ContentType ||
		packageResponse.ContentLength != int64(len(installerBytes)) {
		t.Fatalf(
			"delivered package status/type/length/body = %d/%q/%d/%d, want 200/%q/%d/exact",
			packageResponse.StatusCode,
			packageResponse.Header.Get("Content-Type"),
			packageResponse.ContentLength,
			len(deliveredInstaller),
			installer.ContentType,
			len(installerBytes),
		)
	}

	iconHashesRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/icons/_icon_hashes.plist",
		serial,
	)
	iconHashesResponse, err := munkiClient.Do(iconHashesRequest)
	if err != nil {
		t.Fatalf("fetch first-host Munki icon hashes: %v", err)
	}
	iconHashesBody := readAndClose(t, iconHashesResponse)
	if iconHashesResponse.StatusCode != http.StatusOK {
		t.Fatalf("first-host icon hashes status = %d, want 200", iconHashesResponse.StatusCode)
	}
	var iconHashes map[string]string
	if _, err := plist.Unmarshal(iconHashesBody, &iconHashes); err != nil {
		t.Fatalf("decode first-host Munki icon hashes plist: %v", err)
	}
	if iconHashes[firstIconName] == "" || iconHashes[secondIconName] != "" || len(iconHashes) != 1 {
		t.Fatalf("first-host icon hashes = %+v, want only %q", iconHashes, firstIconName)
	}
	firstIconRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/icons/"+firstIconName,
		serial,
	)
	firstIconResponse, err := munkiClient.Do(firstIconRequest)
	if err != nil {
		t.Fatalf("fetch first-host Munki icon: %v", err)
	}
	if got := readAndClose(t, firstIconResponse); firstIconResponse.StatusCode != http.StatusOK ||
		!bytes.Equal(got, firstIconBytes) {
		t.Fatalf("first-host icon status/body = %d/%d, want 200/exact", firstIconResponse.StatusCode, len(got))
	}
	workerRoot := t.TempDir()
	workerDataDir := filepath.Join(workerRoot, "mirror")
	workerTLS := createTestTLS(t, workerRoot)
	workerPort := allocatePort(t)
	workerBaseURL := "https://localhost:" + strconv.Itoa(workerPort)
	point := createMDPDistributionPoint(t, server, workerBaseURL)
	if point.Key == "" {
		t.Fatal("created distribution point did not reveal its worker key")
	}
	server.redact(point.Key)
	workerLogPath := filepath.Join(workerRoot, "worker.log")
	workerLog, err := os.OpenFile(workerLogPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // Test-owned temporary path.
	if err != nil {
		t.Fatalf("create MDP worker log: %v", err)
	}
	t.Cleanup(func() { _ = workerLog.Close() })
	workerCommand := exec.Command(testBinary(t), "mdp") //nolint:gosec,noctx // E2E harness selects the binary; stopProcess owns shutdown and forced kill.
	workerCommand.Env = append(
		withoutWoodstarEnvironment(os.Environ()),
		"WOODSTAR_MDP_SERVER_URL="+server.BaseURL,
		"WOODSTAR_MDP_SERVER_CA_FILE="+server.CACertificatePath,
		"WOODSTAR_MDP_KEY="+point.Key,
		"WOODSTAR_MDP_DATA_DIR="+workerDataDir,
		"WOODSTAR_MDP_LISTEN_ADDR=127.0.0.1:"+strconv.Itoa(workerPort),
		"WOODSTAR_MDP_TLS_CERT_FILE="+workerTLS.certificatePath,
		"WOODSTAR_MDP_TLS_KEY_FILE="+workerTLS.privateKeyPath,
		"WOODSTAR_MDP_LOG_LEVEL=info",
		"WOODSTAR_MDP_DOWNLOAD_CONCURRENCY=1",
	)
	workerCommand.Stdout = workerLog
	workerCommand.Stderr = workerLog
	workerProcess, err := startProcess(workerCommand)
	if err != nil {
		t.Fatalf("start MDP worker: %v", err)
	}
	t.Cleanup(func() {
		stopProcess(t, "MDP worker", workerProcess)
		if t.Failed() {
			t.Logf("MDP worker logs (tail):\n%s", safeLogTail(workerLogPath, []string{point.Key}))
		}
	})
	if _, err := waitForMDPCurrent(t.Context(), server, point.Id, pkg.Id, workerProcess); err != nil {
		t.Fatalf("wait for MDP package mirror: %v\n%s", err, safeLogTail(workerLogPath, []string{point.Key}))
	}
	secondPackageForFirstHostRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/pkgs/"+secondInstallerItemLocation,
		serial,
	)
	secondPackageForFirstHostResponse, err := munkiClient.Do(secondPackageForFirstHostRequest)
	if err != nil {
		t.Fatalf("fetch second-host package as first host: %v", err)
	}
	drainAndClose(t, secondPackageForFirstHostResponse)
	if secondPackageForFirstHostResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("second-host package as first host status = %d, want 404", secondPackageForFirstHostResponse.StatusCode)
	}
	missingPackageHostResponse, err := server.Admin.GetHostWithResponse(t.Context(), hostID)
	missingPackageHostResponse = requireAPIResponse(
		t,
		"get host after known Munki package miss",
		http.StatusOK,
		missingPackageHostResponse,
		err,
	)
	if missingPackageHostResponse.JSON200 == nil {
		t.Fatal("get host after known Munki package miss returned no JSON body")
	}
	missingPackageHost := *missingPackageHostResponse.JSON200
	missingPackageHeartbeat := requireHeartbeat(t, missingPackageHost.Heartbeats, "munki")
	if len(missingPackageHost.Heartbeats) != 1 ||
		missingPackageHeartbeat.LastSeenAt.Before(manifestHeartbeat.LastSeenAt) ||
		missingPackageHost.LastContact == nil ||
		!missingPackageHost.LastContact.Equal(missingPackageHeartbeat.LastSeenAt) {
		t.Fatalf(
			"host after known Munki package miss = %+v, heartbeat = %+v, want current Munki contact",
			missingPackageHost,
			missingPackageHeartbeat,
		)
	}
	redirectRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/pkgs/"+installerItemLocation,
		serial,
	)
	redirectResponse, err := munkiClient.Do(redirectRequest)
	if err != nil {
		t.Fatalf("request first-host package through MDP: %v", err)
	}
	drainAndClose(t, redirectResponse)
	requireMDPRedirect(t, redirectResponse, workerBaseURL, installerItemLocation)
	secondIconForFirstHostRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/icons/"+secondIconName,
		serial,
	)
	secondIconForFirstHostResponse, err := munkiClient.Do(secondIconForFirstHostRequest)
	if err != nil {
		t.Fatalf("fetch second-host icon as first host: %v", err)
	}
	drainAndClose(t, secondIconForFirstHostResponse)
	if secondIconForFirstHostResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("second-host icon as first host status = %d, want 404", secondIconForFirstHostResponse.StatusCode)
	}
	secondCatalogRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/catalogs/woodstar",
		secondSerial,
	)
	secondCatalogResponse, err := munkiClient.Do(secondCatalogRequest)
	if err != nil {
		t.Fatalf("fetch second-host Munki catalog: %v", err)
	}
	secondCatalogBody := readAndClose(t, secondCatalogResponse)
	if secondCatalogResponse.StatusCode != http.StatusOK {
		t.Fatalf("second-host catalog status = %d, want 200", secondCatalogResponse.StatusCode)
	}
	var secondCatalog []struct {
		Name string `plist:"name"`
	}
	if _, err := plist.Unmarshal(secondCatalogBody, &secondCatalog); err != nil {
		t.Fatalf("decode second-host Munki catalog plist: %v", err)
	}
	if len(secondCatalog) != 1 || secondCatalog[0].Name != secondSoftwareName {
		t.Fatalf("second-host catalog = %+v, want only %q", secondCatalog, secondSoftwareName)
	}
	secondPackageRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/pkgs/"+secondInstallerItemLocation,
		secondSerial,
	)
	secondPackageResponse, err := munkiClient.Do(secondPackageRequest)
	if err != nil {
		t.Fatalf("fetch second-host Munki package: %v", err)
	}
	drainAndClose(t, secondPackageResponse)
	secondRedirectURL := requireMDPRedirect(t, secondPackageResponse, workerBaseURL, secondInstallerItemLocation)
	secondWorkerRequest, err := http.NewRequestWithContext(t.Context(), http.MethodGet, secondRedirectURL, nil)
	if err != nil {
		t.Fatalf("create second-host MDP package request: %v", redactedRequestError(err))
	}
	secondWorkerResponse, err := verifyingClient(t, workerTLS.caCertificate).Do(secondWorkerRequest)
	if err != nil {
		t.Fatalf("fetch second-host package from MDP: %v", redactedRequestError(err))
	}
	if got := readAndClose(t, secondWorkerResponse); secondWorkerResponse.StatusCode != http.StatusOK ||
		!bytes.Equal(got, secondInstallerBytes) {
		t.Fatalf("second-host MDP package status/body = %d/%d, want 200/exact", secondWorkerResponse.StatusCode, len(got))
	}
	secondIconRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/icons/"+secondIconName,
		secondSerial,
	)
	secondIconResponse, err := munkiClient.Do(secondIconRequest)
	if err != nil {
		t.Fatalf("fetch second-host Munki icon: %v", err)
	}
	if got := readAndClose(t, secondIconResponse); secondIconResponse.StatusCode != http.StatusOK ||
		!bytes.Equal(got, secondIconBytes) {
		t.Fatalf("second-host icon status/body = %d/%d, want 200/exact", secondIconResponse.StatusCode, len(got))
	}

	resourcesRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/client_resources/"+serial+".zip",
		serial,
	)
	resourcesResponse, err := munkiClient.Do(resourcesRequest)
	if err != nil {
		t.Fatalf("fetch client resources: %v", err)
	}
	archiveBody := readAndClose(t, resourcesResponse)
	if resourcesResponse.StatusCode != http.StatusOK ||
		resourcesResponse.Header.Get("Content-Type") != "application/zip" ||
		resourcesResponse.ContentLength != int64(len(archiveBody)) {
		t.Fatalf(
			"client resources status/type/length = %d/%q/%d, want 200/application/zip/%d",
			resourcesResponse.StatusCode,
			resourcesResponse.Header.Get("Content-Type"),
			resourcesResponse.ContentLength,
			len(archiveBody),
		)
	}
	archive, err := zip.NewReader(bytes.NewReader(archiveBody), int64(len(archiveBody)))
	if err != nil {
		t.Fatalf("decode delivered client resources ZIP: %v", err)
	}
	archiveFiles := make(map[string][]byte, len(archive.File))
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open client resources %s: %v", file.Name, err)
		}
		body, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil {
			t.Fatalf("read client resources %s: %v", file.Name, readErr)
		}
		if closeErr != nil {
			t.Fatalf("close client resources %s: %v", file.Name, closeErr)
		}
		archiveFiles[file.Name] = body
	}
	if !bytes.Equal(archiveFiles["resources/banner.png"], bannerBytes) {
		t.Fatal("client resources ZIP does not contain the exact uploaded resources/banner.png")
	}
	showcase, ok := archiveFiles["templates/showcase_template.html"]
	if !ok || !strings.Contains(string(showcase), "custom/resources/banner.png") ||
		!strings.Contains(string(showcase), "object-fit: cover") ||
		!strings.Contains(string(showcase), "object-position: 50% center") {
		t.Fatalf("client resources ZIP showcase template = %q, want banner reference", showcase)
	}

	uploadedArchiveBytes := []byte("trusted administrator archive bytes")
	createdArchive, err := server.Admin.CreateMunkiClientResourcesArchiveUploadWithResponse(
		t.Context(),
		adminapi.MunkiDirectUploadRequest{
			Filename: "school-resources.zip",
		},
	)
	createdArchive = requireAPIResponse(
		t,
		"create client resources archive upload",
		http.StatusCreated,
		createdArchive,
		err,
	)
	archiveTarget := createdArchive.JSON201
	if archiveTarget == nil {
		t.Fatal("create client resources archive upload returned no JSON body")
	}
	archiveUploadRequest, err := http.NewRequestWithContext(
		t.Context(),
		string(archiveTarget.Upload.Method),
		archiveTarget.Upload.Url,
		bytes.NewReader(uploadedArchiveBytes),
	)
	if err != nil {
		t.Fatal("create client resources archive capability request")
	}
	if archiveTarget.Upload.Headers != nil {
		for name, value := range *archiveTarget.Upload.Headers {
			archiveUploadRequest.Header.Set(name, value)
		}
	}
	archiveUploadResponse, err := transferClient.Do(archiveUploadRequest)
	if err != nil {
		t.Fatalf("upload client resources archive: %v", err)
	}
	drainAndClose(t, archiveUploadResponse)
	if archiveUploadResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("archive upload status = %d, want %d", archiveUploadResponse.StatusCode, http.StatusNoContent)
	}

	updatedArchive, err := server.Admin.UpdateMunkiClientResourcesWithResponse(
		t.Context(),
		clientResources.Id,
		adminapi.MunkiClientResourcesMutation{ArchiveObjectId: new(archiveTarget.ObjectId)},
	)
	updatedArchive = requireAPIResponse(
		t,
		"update client resources archive",
		http.StatusOK,
		updatedArchive,
		err,
	)
	if updatedArchive.JSON200 == nil {
		t.Fatal("update client resources archive returned no JSON body")
	}
	uploadedResources := *updatedArchive.JSON200
	if !uploadedResources.Custom || uploadedResources.Builder == nil ||
		uploadedResources.Builder.Banner.Id != bannerTarget.ObjectId ||
		uploadedResources.Archive.Id != archiveTarget.ObjectId ||
		uploadedResources.Archive.Filename != "school-resources.zip" ||
		uploadedResources.Archive.SizeBytes == nil ||
		*uploadedResources.Archive.SizeBytes != int64(len(uploadedArchiveBytes)) {
		t.Fatalf("uploaded client resources = %+v", uploadedResources)
	}

	uploadedAdminResponse := getResponse(
		t,
		server.AdminHTTP,
		server.BaseURL+uploadedResources.Archive.ContentUrl,
	)
	if got := readAndClose(t, uploadedAdminResponse); uploadedAdminResponse.StatusCode != http.StatusOK ||
		!bytes.Equal(got, uploadedArchiveBytes) {
		t.Fatalf(
			"uploaded archive admin content status/body = %d/%q, want 200/exact",
			uploadedAdminResponse.StatusCode,
			got,
		)
	}
	retainedBannerResponse := getResponse(
		t,
		server.AdminHTTP,
		server.BaseURL+rereadClientResources.Builder.Banner.ContentUrl,
	)
	if got := readAndClose(t, retainedBannerResponse); retainedBannerResponse.StatusCode != http.StatusOK ||
		!bytes.Equal(got, bannerBytes) {
		t.Fatalf(
			"retained builder banner status/body = %d/%q, want 200/exact",
			retainedBannerResponse.StatusCode,
			got,
		)
	}

	uploadedResourcesRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/client_resources/"+serial+".zip",
		serial,
	)
	uploadedResourcesResponse, err := munkiClient.Do(uploadedResourcesRequest)
	if err != nil {
		t.Fatalf("fetch uploaded client resources: %v", err)
	}
	if got := readAndClose(t, uploadedResourcesResponse); uploadedResourcesResponse.StatusCode != http.StatusOK ||
		!bytes.Equal(got, uploadedArchiveBytes) {
		t.Fatalf(
			"uploaded client resources status/body = %d/%q, want 200/exact",
			uploadedResourcesResponse.StatusCode,
			got,
		)
	}

	rebuiltBuilder, err := server.Admin.UpdateMunkiClientResourcesWithResponse(
		t.Context(),
		clientResources.Id,
		adminapi.MunkiClientResourcesMutation{Builder: &builder},
	)
	rebuiltBuilder = requireAPIResponse(
		t,
		"rebuild retained client resources builder",
		http.StatusOK,
		rebuiltBuilder,
		err,
	)
	if rebuiltBuilder.JSON200 == nil || rebuiltBuilder.JSON200.Custom ||
		rebuiltBuilder.JSON200.Builder == nil ||
		rebuiltBuilder.JSON200.Builder.Banner.Id != bannerTarget.ObjectId {
		t.Fatalf("rebuilt client resources = %+v", rebuiltBuilder.JSON200)
	}
	replacedArchiveResponse := getResponse(
		t,
		server.AdminHTTP,
		server.BaseURL+uploadedResources.Archive.ContentUrl,
	)
	drainAndClose(t, replacedArchiveResponse)
	if replacedArchiveResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("replaced custom archive status = %d, want 404", replacedArchiveResponse.StatusCode)
	}

	deletedResources, err := server.Admin.DeleteMunkiClientResourcesWithResponse(
		t.Context(),
		clientResources.Id,
	)
	requireAPIResponse(t, "undeploy client resources", http.StatusNoContent, deletedResources, err)
	undeployedResources, err := server.Admin.ListMunkiClientResourcesWithResponse(t.Context(), nil)
	undeployedResources = requireAPIResponse(
		t,
		"list undeployed client resources",
		http.StatusOK,
		undeployedResources,
		err,
	)
	if undeployedResources.JSON200 == nil || undeployedResources.JSON200.Count != 0 ||
		len(undeployedResources.JSON200.Items) != 0 {
		t.Fatalf("undeployed client resources = %+v, want empty page", undeployedResources.JSON200)
	}
	missingResources, err := server.Admin.GetMunkiClientResourcesWithResponse(
		t.Context(),
		clientResources.Id,
	)
	missingResources = requireAPIResponse(
		t,
		"get deleted client resources",
		http.StatusNotFound,
		missingResources,
		err,
	)
	if missingResources.ApplicationproblemJSON404 == nil ||
		missingResources.ApplicationproblemJSON404.Detail != nil {
		t.Fatalf(
			"deleted client resources error = %+v, want 404 without detail",
			missingResources.ApplicationproblemJSON404,
		)
	}

	undeployedRequest := newMunkiRequest(
		t,
		t.Context(),
		server.BaseURL+"/munki/client_resources/"+serial+".zip",
		serial,
	)
	undeployedResponse, err := munkiClient.Do(undeployedRequest)
	if err != nil {
		t.Fatalf("fetch undeployed client resources: %v", err)
	}
	drainAndClose(t, undeployedResponse)
	if undeployedResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("undeployed client resources status = %d, want 404", undeployedResponse.StatusCode)
	}
}

func newMunkiRequest(
	t *testing.T,
	ctx context.Context,
	url string,
	serial string,
) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("create Munki request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+munkiSecret)
	req.Header.Set("X-Woodstar-Serial-Number", serial)
	return req
}

func attachMunkiIcon(
	t *testing.T,
	server *testServer,
	transferClient *http.Client,
	softwareID int64,
	filename string,
	contents []byte,
) adminapi.MunkiObjectView {
	t.Helper()
	payload, err := json.Marshal(adminapi.MunkiDirectUploadRequest{
		Filename: filename,
	})
	if err != nil {
		t.Fatalf("encode icon upload request: %v", err)
	}
	createRequest, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		server.BaseURL+"/api/munki/icons",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("create icon upload request: %v", err)
	}
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse, err := server.AdminHTTP.Do(createRequest)
	if err != nil {
		t.Fatalf("create icon upload: %v", err)
	}
	createBody := readAndClose(t, createResponse)
	if createResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create icon upload status = %d, want 201: %s", createResponse.StatusCode, createBody)
	}
	var target adminapi.MunkiDirectUploadTarget
	if err := json.Unmarshal(createBody, &target); err != nil {
		t.Fatalf("decode icon upload target: %v", err)
	}
	if target.ObjectId <= 0 || target.Upload.Method != http.MethodPut || target.Upload.Strategy != "direct-put" {
		t.Fatalf(
			"icon upload target id/method/strategy = %d/%q/%q, want positive/PUT/direct-put",
			target.ObjectId,
			target.Upload.Method,
			target.Upload.Strategy,
		)
	}
	uploadRequest, err := http.NewRequestWithContext(
		t.Context(),
		string(target.Upload.Method),
		target.Upload.Url,
		bytes.NewReader(contents),
	)
	if err != nil {
		t.Fatalf("create icon upload capability request: %v", err)
	}
	if target.Upload.Headers != nil {
		for name, value := range *target.Upload.Headers {
			uploadRequest.Header.Set(name, value)
		}
	}
	uploadResponse, err := transferClient.Do(uploadRequest)
	if err != nil {
		t.Fatalf("upload icon through returned capability: %v", err)
	}
	drainAndClose(t, uploadResponse)
	if uploadResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("icon upload status = %d, want 204", uploadResponse.StatusCode)
	}
	attachPayload, err := json.Marshal(struct {
		ObjectID int64 `json:"object_id"`
	}{ObjectID: target.ObjectId})
	if err != nil {
		t.Fatalf("encode attach icon request: %v", err)
	}
	attachRequest, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		server.BaseURL+"/api/munki/software/"+strconv.FormatInt(softwareID, 10)+"/icon",
		bytes.NewReader(attachPayload),
	)
	if err != nil {
		t.Fatalf("create attach icon request: %v", err)
	}
	attachRequest.Header.Set("Content-Type", "application/json")
	attachResponse, err := server.AdminHTTP.Do(attachRequest)
	if err != nil {
		t.Fatalf("attach icon: %v", err)
	}
	attachBody := readAndClose(t, attachResponse)
	if attachResponse.StatusCode != http.StatusOK {
		t.Fatalf("attach icon status = %d, want 200: %s", attachResponse.StatusCode, attachBody)
	}
	var icon adminapi.MunkiObjectView
	if err := json.Unmarshal(attachBody, &icon); err != nil {
		t.Fatalf("decode attached icon: %v", err)
	}
	if icon.Id != target.ObjectId || icon.Filename != filename || icon.ContentType != "image/png" ||
		icon.SizeBytes == nil || *icon.SizeBytes != int64(len(contents)) || icon.Sha256 == nil || *icon.Sha256 == "" {
		t.Fatalf("attached icon = %+v, want finalized %s", icon, filename)
	}
	return icon
}

func assertStorageCapabilityTTL(
	t *testing.T,
	rawURL string,
	keyHex string,
	op string,
	issuedAfter time.Time,
	issuedBefore time.Time,
) {
	t.Helper()
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		t.Fatalf("decode storage capability key: %v", err)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse storage capability URL: %v", err)
	}
	claims, err := capability.Verify[storage.BlobCapabilityClaims](
		key,
		parsed.Query().Get("cap"),
		op,
		issuedAfter,
	)
	if err != nil {
		t.Fatalf("verify storage capability: %v", err)
	}
	minExpiry := issuedAfter.Add(testStorageTransferTTL).Unix()
	maxExpiry := issuedBefore.Add(testStorageTransferTTL).Unix()
	if claims.Exp < minExpiry || claims.Exp > maxExpiry {
		t.Fatalf("storage capability expiry = %d, want between %d and %d", claims.Exp, minExpiry, maxExpiry)
	}
}
