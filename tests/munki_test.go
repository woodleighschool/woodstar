package integration

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
)

const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

type munkiTestUploadTarget struct {
	ObjectID int64                 `json:"object_id"`
	Upload   munkiTestUploadAction `json:"upload"`
}

type munkiTestUploadAction struct {
	Strategy string            `json:"strategy"`
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
}

type munkiTestObject struct {
	ID          int64   `json:"id"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	SizeBytes   *int64  `json:"size_bytes"`
	SHA256      *string `json:"sha256"`
	ContentURL  string  `json:"content_url"`
}

type munkiTestInstallerFile struct {
	Filename              string `json:"filename"`
	InstallerItemLocation string `json:"installer_item_location"`
	SizeBytes             int64  `json:"size_bytes"`
	SHA256                string `json:"sha256"`
}

type munkiTestPackage struct {
	ID                int64                    `json:"id"`
	Software          munkiTestPackageSoftware `json:"software"`
	Version           string                   `json:"version"`
	InstallerType     string                   `json:"installer_type"`
	InstallerObjectID *int64                   `json:"installer_object_id"`
	InstallerFile     *munkiTestInstallerFile  `json:"installer_file"`
}

type munkiTestPackageSoftware struct {
	ID int64 `json:"id"`
}

type munkiTestPackageSelector struct {
	Strategy string `json:"strategy"`
}

type munkiTestInclude struct {
	LabelID int64                    `json:"label_id"`
	Package munkiTestPackageSelector `json:"package"`
	Actions []string                 `json:"actions"`
}

type munkiTestLabelRef struct {
	LabelID int64 `json:"label_id"`
}

type munkiTestTargets struct {
	Include []munkiTestInclude  `json:"include"`
	Exclude []munkiTestLabelRef `json:"exclude"`
}

type munkiTestSoftware struct {
	ID       int64              `json:"id"`
	Name     string             `json:"name"`
	Packages []munkiTestPackage `json:"packages"`
	Targets  munkiTestTargets   `json:"targets"`
}

type munkiTestLink struct {
	Label         string `json:"label"`
	Target        string `json:"target"`
	OpenInBrowser bool   `json:"open_in_browser"`
}

type munkiTestClientResources struct {
	Banner          munkiTestObject `json:"banner"`
	BannerAlignment string          `json:"banner_alignment"`
	Links           []munkiTestLink `json:"links"`
	FooterText      string          `json:"footer_text"`
	FooterLinks     []munkiTestLink `json:"footer_links"`
}

func TestMunki(t *testing.T) {
	const (
		serial       = "C02WOODSTARMUNKI"
		softwareName = "WoodstarIntegrationApp"
		munkiSecret  = "munki-integration-secret-0123456789abcdef"
	)

	server := startTestServer(t)
	server.redact(munkiSecret)
	transferClient := verifyingClient(t, server.CACertificate)

	var setupUser struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	setupResponse := requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/setup",
		struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Password string `json:"password"`
		}{
			Email:    "admin@woodstar.test",
			Name:     "Integration Administrator",
			Password: "integration-admin-password",
		},
		http.StatusCreated,
		&setupUser,
	)
	if setupUser.ID <= 0 || setupUser.Email != "admin@woodstar.test" {
		t.Fatalf("setup user = %+v, want created integration administrator", setupUser)
	}
	secureCookie := false
	for _, cookie := range setupResponse.Cookies() {
		secureCookie = secureCookie || cookie.Secure
	}
	if !secureCookie {
		t.Fatal("setup response did not issue a secure session cookie")
	}
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

	var createdSecret struct {
		ID    int64  `json:"id"`
		Agent string `json:"agent"`
	}
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/agent-secrets",
		struct {
			Agent string `json:"agent"`
			Value string `json:"value"`
		}{Agent: "munki", Value: munkiSecret},
		http.StatusCreated,
		&createdSecret,
	)
	if createdSecret.ID <= 0 || createdSecret.Agent != "munki" {
		t.Fatalf("created agent secret = %+v, want active Munki secret", createdSecret)
	}

	installerBytes := bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, 200)
	installerSum := sha256.Sum256(installerBytes)
	installerSHA256 := hex.EncodeToString(installerSum[:])
	var installerTarget munkiTestUploadTarget
	capabilityIssuedAfter := time.Now()
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/munki/package-installers",
		struct {
			Filename string `json:"filename"`
		}{Filename: "WoodstarIntegration.pkg"},
		http.StatusCreated,
		&installerTarget,
	)
	capabilityIssuedBefore := time.Now()
	if installerTarget.ObjectID <= 0 || installerTarget.Upload.Method != http.MethodPut ||
		installerTarget.Upload.Strategy != "direct-put" {
		t.Fatalf(
			"installer upload target id/method/strategy = %d/%q/%q, want positive/PUT/direct-put",
			installerTarget.ObjectID,
			installerTarget.Upload.Method,
			installerTarget.Upload.Strategy,
		)
	}
	assertStorageCapabilityTTL(
		t,
		installerTarget.Upload.URL,
		server.StorageCapabilityKey,
		capability.OpPut,
		capabilityIssuedAfter,
		capabilityIssuedBefore,
	)
	installerUpload, err := http.NewRequestWithContext(
		t.Context(),
		installerTarget.Upload.Method,
		installerTarget.Upload.URL,
		bytes.NewReader(installerBytes),
	)
	if err != nil {
		t.Fatal("create installer upload capability request")
	}
	for name, value := range installerTarget.Upload.Headers {
		installerUpload.Header.Set(name, value)
	}
	installerUploadResponse, err := transferClient.Do(installerUpload)
	if err != nil {
		t.Fatal("upload installer through returned capability")
	}
	drainAndClose(t, installerUploadResponse)
	if installerUploadResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("installer upload status = %d, want %d", installerUploadResponse.StatusCode, http.StatusNoContent)
	}

	var installer munkiTestObject
	requestJSON(
		t,
		server.Client,
		http.MethodPut,
		server.BaseURL+"/api/munki/package-installers/"+strconv.FormatInt(installerTarget.ObjectID, 10),
		nil,
		http.StatusOK,
		&installer,
	)
	if installer.ID != installerTarget.ObjectID || installer.Filename != "WoodstarIntegration.pkg" ||
		installer.ContentType != "application/octet-stream" || installer.SizeBytes == nil ||
		*installer.SizeBytes != int64(len(installerBytes)) || installer.SHA256 == nil ||
		*installer.SHA256 != installerSHA256 ||
		installer.ContentURL != "/api/munki/package-installers/"+
			strconv.FormatInt(installer.ID, 10)+"/content" {
		t.Fatal("finalized installer did not contain the expected server-derived metadata")
	}
	installerContentResponse, err := server.Client.Get(server.BaseURL + installer.ContentURL)
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
		server.BaseURL+installer.ContentURL,
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

	targets := munkiTestTargets{
		Include: []munkiTestInclude{{
			LabelID: allHostsLabelID,
			Package: munkiTestPackageSelector{Strategy: "latest"},
			Actions: []string{"managed_installs"},
		}},
		Exclude: []munkiTestLabelRef{},
	}
	var software munkiTestSoftware
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/munki/software",
		struct {
			Name        string           `json:"name"`
			DisplayName string           `json:"display_name"`
			Description string           `json:"description"`
			Category    string           `json:"category"`
			Developer   string           `json:"developer"`
			Targets     munkiTestTargets `json:"targets"`
		}{
			Name:        softwareName,
			DisplayName: "Woodstar Integration App",
			Description: "Compiled Munki repository lifecycle fixture.",
			Category:    "Testing",
			Developer:   "Woodleigh School",
			Targets:     targets,
		},
		http.StatusCreated,
		&software,
	)
	if software.ID <= 0 || software.Name != softwareName {
		t.Fatalf("created software = %+v, want %s", software, softwareName)
	}

	var pkg munkiTestPackage
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/munki/packages",
		struct {
			SoftwareID        int64  `json:"software_id"`
			Version           string `json:"version"`
			InstallerType     string `json:"installer_type"`
			InstallerObjectID int64  `json:"installer_object_id"`
		}{
			SoftwareID:        software.ID,
			Version:           "1.0",
			InstallerType:     "pkg",
			InstallerObjectID: installer.ID,
		},
		http.StatusCreated,
		&pkg,
	)
	installerItemLocation := fmt.Sprintf(
		"packages/%d/installer/%s",
		pkg.ID,
		installer.Filename,
	)
	if pkg.Software.ID != software.ID || pkg.Version != "1.0" || pkg.InstallerType != "pkg" ||
		pkg.InstallerObjectID == nil || *pkg.InstallerObjectID != installer.ID ||
		pkg.InstallerFile == nil || pkg.InstallerFile.Filename != installer.Filename ||
		pkg.InstallerFile.InstallerItemLocation != installerItemLocation ||
		pkg.InstallerFile.SizeBytes != int64(len(installerBytes)) ||
		pkg.InstallerFile.SHA256 != installerSHA256 {
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
	var bannerTarget munkiTestUploadTarget
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/munki/client-resources/banner",
		struct {
			Filename string `json:"filename"`
		}{Filename: "banner.png"},
		http.StatusCreated,
		&bannerTarget,
	)
	if bannerTarget.ObjectID <= 0 || bannerTarget.Upload.Method != http.MethodPut ||
		bannerTarget.Upload.Strategy != "direct-put" {
		t.Fatalf(
			"banner upload target id/method/strategy = %d/%q/%q, want positive/PUT/direct-put",
			bannerTarget.ObjectID,
			bannerTarget.Upload.Method,
			bannerTarget.Upload.Strategy,
		)
	}
	bannerUpload, err := http.NewRequestWithContext(
		t.Context(),
		bannerTarget.Upload.Method,
		bannerTarget.Upload.URL,
		bytes.NewReader(bannerBytes),
	)
	if err != nil {
		t.Fatal("create banner upload capability request")
	}
	for name, value := range bannerTarget.Upload.Headers {
		bannerUpload.Header.Set(name, value)
	}
	bannerUploadResponse, err := transferClient.Do(bannerUpload)
	if err != nil {
		t.Fatal("upload banner through returned capability")
	}
	drainAndClose(t, bannerUploadResponse)
	if bannerUploadResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("banner upload status = %d, want %d", bannerUploadResponse.StatusCode, http.StatusNoContent)
	}

	links := []munkiTestLink{{
		Label:         "Support",
		Target:        "https://support.woodstar.test/",
		OpenInBrowser: true,
	}}
	footerLinks := []munkiTestLink{{
		Label:  "Updates",
		Target: "munki://updates",
	}}
	var clientResources munkiTestClientResources
	requestJSON(
		t,
		server.Client,
		http.MethodPut,
		server.BaseURL+"/api/munki/client-resources",
		struct {
			BannerObjectID  int64           `json:"banner_object_id"`
			BannerAlignment string          `json:"banner_alignment"`
			Links           []munkiTestLink `json:"links"`
			FooterText      string          `json:"footer_text"`
			FooterLinks     []munkiTestLink `json:"footer_links"`
		}{
			BannerObjectID:  bannerTarget.ObjectID,
			BannerAlignment: "center",
			Links:           links,
			FooterText:      "Managed by Woodstar",
			FooterLinks:     footerLinks,
		},
		http.StatusOK,
		&clientResources,
	)
	bannerSum := sha256.Sum256(bannerBytes)
	bannerSHA256 := hex.EncodeToString(bannerSum[:])
	if clientResources.Banner.ID != bannerTarget.ObjectID ||
		clientResources.Banner.ContentType != "image/png" ||
		clientResources.Banner.SizeBytes == nil ||
		*clientResources.Banner.SizeBytes != int64(len(bannerBytes)) ||
		clientResources.Banner.SHA256 == nil || *clientResources.Banner.SHA256 != bannerSHA256 ||
		clientResources.BannerAlignment != "center" || clientResources.FooterText != "Managed by Woodstar" {
		t.Fatal("saved client resources did not contain the expected compiled banner state")
	}

	var rereadSoftware munkiTestSoftware
	requestJSON(
		t,
		server.Client,
		http.MethodGet,
		server.BaseURL+"/api/munki/software/"+strconv.FormatInt(software.ID, 10),
		nil,
		http.StatusOK,
		&rereadSoftware,
	)
	if rereadSoftware.ID != software.ID || len(rereadSoftware.Packages) != 1 ||
		rereadSoftware.Packages[0].ID != pkg.ID || len(rereadSoftware.Targets.Include) != 1 ||
		rereadSoftware.Targets.Include[0].LabelID != allHostsLabelID ||
		rereadSoftware.Targets.Include[0].Package.Strategy != "latest" ||
		len(rereadSoftware.Targets.Include[0].Actions) != 1 ||
		rereadSoftware.Targets.Include[0].Actions[0] != "managed_installs" {
		t.Fatalf("re-read software = %+v, want saved package and all-hosts target", rereadSoftware)
	}
	var rereadClientResources munkiTestClientResources
	requestJSON(
		t,
		server.Client,
		http.MethodGet,
		server.BaseURL+"/api/munki/client-resources",
		nil,
		http.StatusOK,
		&rereadClientResources,
	)
	if rereadClientResources.Banner.ID != bannerTarget.ObjectID ||
		rereadClientResources.Banner.ContentURL != "/api/munki/client-resources/banner/"+
			strconv.FormatInt(bannerTarget.ObjectID, 10)+"/content" ||
		rereadClientResources.BannerAlignment != "center" ||
		len(rereadClientResources.Links) != 1 || rereadClientResources.Links[0] != links[0] ||
		len(rereadClientResources.FooterLinks) != 1 ||
		rereadClientResources.FooterLinks[0] != footerLinks[0] {
		t.Fatal("re-read client resources did not match the saved public state")
	}
	bannerContentResponse, err := server.Client.Get(
		server.BaseURL + rereadClientResources.Banner.ContentURL,
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
