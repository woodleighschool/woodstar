package e2e

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/url"
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

const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

func TestMunki(t *testing.T) {
	const (
		serial       = "C02WOODSTARMUNKI"
		softwareName = "WoodstarIntegrationApp"
		munkiSecret  = "munki-integration-secret-0123456789abcdef"
	)

	server := startTestServer(t)
	server.redact(munkiSecret)
	transferClient := verifyingClient(t, server.CACertificate)

	setupAdmin(
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
		t.Fatal("admin client did not retain the setup session cookie")
	}

	database, err := pgx.Connect(t.Context(), server.DatabaseURL)
	if err != nil {
		t.Fatalf("connect to isolated Woodstar database: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), databaseOperationTimeout)
		defer cancel()
		if closeErr := database.Close(ctx); closeErr != nil {
			t.Errorf("close isolated Woodstar database connection: %v", closeErr)
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
	createdInstaller, err := server.Admin.CreateMunkiPackageInstallerWithResponse(
		t.Context(),
		adminapi.MunkiUploadRequest{Filename: "WoodstarIntegration.pkg"},
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
	installerUploadAction := directUpload(t, installerTarget)
	if installerTarget.ObjectId <= 0 || installerUploadAction.Method != http.MethodPut ||
		installerUploadAction.Strategy != "direct-put" {
		t.Fatalf(
			"installer upload target id/method/strategy = %d/%q/%q, want positive/PUT/direct-put",
			installerTarget.ObjectId,
			installerUploadAction.Method,
			installerUploadAction.Strategy,
		)
	}
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
		installerUploadAction.Method,
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

	finalizedInstaller, err := server.Admin.FinalizeMunkiPackageInstallerWithResponse(
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
	installerContentResponse, err := server.AdminHTTP.Get(server.BaseURL + installer.ContentUrl)
	if err != nil {
		t.Fatalf("fetch installer through admin content route: %v", err)
	}
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
			Developer:   new("Woodleigh School"),
			Targets:     targets,
		},
	)
	createdSoftware = requireAPIResponse(t, "create software", http.StatusCreated, createdSoftware, err)
	if createdSoftware.JSON201 == nil || createdSoftware.JSON201.Id <= 0 ||
		createdSoftware.JSON201.Name != softwareName {
		t.Fatalf("created software = %+v, want %s", createdSoftware.JSON201, softwareName)
	}
	software := *createdSoftware.JSON201

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
		adminapi.MunkiUploadRequest{Filename: "banner.png"},
	)
	createdBanner = requireAPIResponse(t, "create banner upload", http.StatusCreated, createdBanner, err)
	bannerTarget := createdBanner.JSON201
	bannerUploadAction := directUpload(t, bannerTarget)
	if bannerTarget.ObjectId <= 0 || bannerUploadAction.Method != http.MethodPut ||
		bannerUploadAction.Strategy != "direct-put" {
		t.Fatalf(
			"banner upload target id/method/strategy = %d/%q/%q, want positive/PUT/direct-put",
			bannerTarget.ObjectId,
			bannerUploadAction.Method,
			bannerUploadAction.Strategy,
		)
	}
	bannerUpload, err := http.NewRequestWithContext(
		t.Context(),
		bannerUploadAction.Method,
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
	savedResources, err := server.Admin.SaveMunkiClientResourcesWithResponse(
		t.Context(),
		adminapi.MunkiMutation{
			BannerObjectId:  bannerTarget.ObjectId,
			BannerAlignment: "center",
			Links:           links,
			FooterText:      "Managed by Woodstar",
			FooterLinks:     footerLinks,
		},
	)
	savedResources = requireAPIResponse(t, "save client resources", http.StatusOK, savedResources, err)
	if savedResources.JSON200 == nil {
		t.Fatal("save client resources returned no JSON body")
	}
	clientResources := *savedResources.JSON200
	bannerSum := sha256.Sum256(bannerBytes)
	bannerSHA256 := hex.EncodeToString(bannerSum[:])
	if clientResources.Banner.Id != bannerTarget.ObjectId ||
		clientResources.Banner.ContentType != "image/png" ||
		clientResources.Banner.SizeBytes == nil ||
		*clientResources.Banner.SizeBytes != int64(len(bannerBytes)) ||
		clientResources.Banner.Sha256 == nil || *clientResources.Banner.Sha256 != bannerSHA256 ||
		clientResources.BannerAlignment != "center" || clientResources.FooterText != "Managed by Woodstar" {
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
	rereadResourcesResponse, err := server.Admin.GetMunkiClientResourcesWithResponse(t.Context())
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
	if rereadClientResources.Banner.Id != bannerTarget.ObjectId ||
		rereadClientResources.Banner.ContentUrl != "/api/munki/client-resources/banner/"+
			strconv.FormatInt(bannerTarget.ObjectId, 10)+"/content" ||
		rereadClientResources.BannerAlignment != "center" ||
		len(rereadClientResources.Links) != 1 || rereadClientResources.Links[0] != links[0] ||
		len(rereadClientResources.FooterLinks) != 1 ||
		rereadClientResources.FooterLinks[0] != footerLinks[0] {
		t.Fatal("re-read client resources did not match the saved public state")
	}
	bannerContentResponse, err := server.AdminHTTP.Get(
		server.BaseURL + rereadClientResources.Banner.ContentUrl,
	)
	if err != nil {
		t.Fatalf("fetch banner through admin content route: %v", err)
	}
	if got := readAndClose(t, bannerContentResponse); bannerContentResponse.StatusCode != http.StatusOK ||
		!bytes.Equal(got, bannerBytes) {
		t.Fatalf("admin banner content status/body = %d/%d, want 200/exact", bannerContentResponse.StatusCode, len(got))
	}

	munkiClient := verifyingClient(t, server.CACertificate)
	munkiClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	manifestRequest, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		server.BaseURL+"/munki/manifests/"+serial,
		nil,
	)
	if err != nil {
		t.Fatalf("create manifest request: %v", err)
	}
	manifestRequest.Header.Set("Authorization", "Bearer "+munkiSecret)
	manifestResponse, err := munkiClient.Do(manifestRequest)
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

	cachedManifestRequest, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		server.BaseURL+"/munki/manifests/"+serial,
		nil,
	)
	if err != nil {
		t.Fatalf("create cached manifest request: %v", err)
	}
	cachedManifestRequest.Header.Set("Authorization", "Bearer "+munkiSecret)
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

	catalogRequest, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		server.BaseURL+"/munki/catalogs/woodstar",
		nil,
	)
	if err != nil {
		t.Fatalf("create catalog request: %v", err)
	}
	catalogRequest.Header.Set("Authorization", "Bearer "+munkiSecret)
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
	}
	if _, err := plist.Unmarshal(catalogBody, &catalog); err != nil {
		t.Fatalf("decode Munki catalog plist: %v", err)
	}
	wantInstallerKiB := (int64(len(installerBytes)) + 1023) / 1024
	if len(catalog) != 1 || catalog[0].Name != softwareName || catalog[0].Version != "1.0" ||
		catalog[0].InstallerItemLocation != installerItemLocation ||
		catalog[0].InstallerItemHash != installerSHA256 ||
		catalog[0].InstallerItemSize != wantInstallerKiB {
		t.Fatalf("catalog = %+v, want package %s at %s", catalog, softwareName, installerItemLocation)
	}

	packageRequest, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		server.BaseURL+"/munki/pkgs/"+installerItemLocation,
		nil,
	)
	if err != nil {
		t.Fatalf("create package request: %v", err)
	}
	packageRequest.Header.Set("Authorization", "Bearer "+munkiSecret)
	sessionOnlyPackageRequest, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		server.BaseURL+"/munki/pkgs/"+installerItemLocation,
		nil,
	)
	if err != nil {
		t.Fatal("create session-only package request")
	}
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

	resourcesRequest, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		server.BaseURL+"/munki/client_resources/"+serial+".zip",
		nil,
	)
	if err != nil {
		t.Fatalf("create client resources request: %v", err)
	}
	resourcesRequest.Header.Set("Authorization", "Bearer "+munkiSecret)
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
	if !ok || !strings.Contains(string(showcase), "custom/resources/banner.png") {
		t.Fatalf("client resources ZIP showcase template = %q, want banner reference", showcase)
	}
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
