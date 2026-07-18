package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
)

const mdpLifecycleTimeout = 30 * time.Second

type mdpUploadTarget struct {
	ObjectID int64           `json:"object_id"`
	Upload   mdpUploadAction `json:"upload"`
}

type mdpUploadAction struct {
	Strategy string            `json:"strategy"`
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
}

type mdpObject struct {
	ID       int64  `json:"id"`
	Filename string `json:"filename"`
}

type mdpInstallerFile struct {
	Filename              string `json:"filename"`
	InstallerItemLocation string `json:"installer_item_location"`
}

type mdpPackage struct {
	ID            int64              `json:"id"`
	Software      mdpPackageSoftware `json:"software"`
	InstallerFile *mdpInstallerFile  `json:"installer_file"`
}

type mdpPackageSoftware struct {
	ID int64 `json:"id"`
}

type mdpPackageState struct {
	PackageID int64  `json:"package_id"`
	Status    string `json:"status"`
}

type mdpDistributionPoint struct {
	ID            int64             `json:"id"`
	Name          string            `json:"name"`
	Enabled       bool              `json:"enabled"`
	ClientBaseURL string            `json:"client_base_url"`
	Online        bool              `json:"online"`
	Key           string            `json:"key"`
	Packages      []mdpPackageState `json:"packages"`
}

type mdpEmptyTargets struct {
	Include []struct{} `json:"include"`
	Exclude []struct{} `json:"exclude"`
}

func TestMDP(t *testing.T) {
	const (
		installerFilename = "WoodstarMDPIntegration.pkg"
		munkiSecret       = "munki-mdp-integration-secret-0123456789"
	)

	server := startTestServer(t)
	server.redact(munkiSecret)
	setupMDPAdmin(t, server)
	createMDPMunkiSecret(t, server, munkiSecret)

	installerBytes := bytes.Repeat(
		[]byte("woodstar-mdp-storage-agnostic-installer\x00\x01\x02\x03"),
		128,
	)
	installer := createMDPInstaller(t, server, installerFilename, installerBytes)
	softwareID := createMDPSoftware(t, server)
	pkg := createMDPPackage(t, server, softwareID, installer)

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
	workerLog, err := os.OpenFile(workerLogPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("create MDP worker log: %v", err)
	}
	t.Cleanup(func() { _ = workerLog.Close() })

	workerCommand := exec.Command(testBinary(t), "mdp")
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
			t.Logf(
				"MDP worker logs (tail):\n%s",
				safeLogTail(workerLogPath, []string{point.Key}),
			)
		}
	})

	detail, err := waitForMDPCurrent(
		t.Context(),
		server,
		point.ID,
		pkg.ID,
		workerProcess,
	)
	if err != nil {
		t.Fatalf(
			"wait for MDP package mirror: %v\n%s",
			err,
			safeLogTail(workerLogPath, []string{point.Key}),
		)
	}
	if !detail.Enabled || detail.Name != "Local integration MDP" ||
		detail.ClientBaseURL != workerBaseURL {
		t.Fatalf(
			"distribution point detail id/name/enabled/client URL = %d/%q/%t/%q, want enabled local integration MDP",
			detail.ID,
			detail.Name,
			detail.Enabled,
			detail.ClientBaseURL,
		)
	}

	mirroredPath := filepath.Join(workerDataDir, fmt.Sprintf("%d-%s", pkg.ID, installerFilename))
	mirroredBytes, err := os.ReadFile(mirroredPath)
	if err != nil {
		t.Fatalf("read mirrored installer: %v", err)
	}
	if !bytes.Equal(mirroredBytes, installerBytes) {
		t.Fatalf(
			"mirrored installer bytes = %d bytes, want exact %d-byte upload",
			len(mirroredBytes),
			len(installerBytes),
		)
	}

	redirectClient := verifyingClient(t, server.CACertificate)
	redirectClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	packageRequest, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		server.BaseURL+"/munki/pkgs/"+pkg.InstallerFile.InstallerItemLocation,
		nil,
	)
	if err != nil {
		t.Fatalf("create Munki package request: %v", err)
	}
	packageRequest.Header.Set("Authorization", "Bearer "+munkiSecret)
	redirectResponse, err := redirectClient.Do(packageRequest)
	if err != nil {
		t.Fatalf("request Munki package: %v", err)
	}
	drainAndClose(t, redirectResponse)
	redirectURL := requireMDPRedirect(t, redirectResponse, workerBaseURL, pkg.InstallerFile.InstallerItemLocation)

	workerClient := verifyingClient(t, workerTLS.caCertificate)
	fullRequest, err := http.NewRequestWithContext(t.Context(), http.MethodGet, redirectURL, nil)
	if err != nil {
		t.Fatalf("create MDP download request: %v", redactedRequestError(err))
	}
	fullResponse, err := workerClient.Do(fullRequest)
	if err != nil {
		t.Fatalf("download package from MDP: %v", redactedRequestError(err))
	}
	fullBody := readAndClose(t, fullResponse)
	if fullResponse.StatusCode != http.StatusOK || !bytes.Equal(fullBody, installerBytes) ||
		fullResponse.ContentLength != int64(len(installerBytes)) {
		t.Fatalf(
			"MDP full response status/length/body = %d/%d/%d bytes, want 200/%d/exact",
			fullResponse.StatusCode,
			fullResponse.ContentLength,
			len(fullBody),
			len(installerBytes),
		)
	}

	const rangeStart, rangeEnd = 37, 113
	rangeRequest, err := http.NewRequestWithContext(t.Context(), http.MethodGet, redirectURL, nil)
	if err != nil {
		t.Fatalf("create MDP range request: %v", redactedRequestError(err))
	}
	rangeRequest.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))
	rangeResponse, err := workerClient.Do(rangeRequest)
	if err != nil {
		t.Fatalf("download package range from MDP: %v", redactedRequestError(err))
	}
	rangeBody := readAndClose(t, rangeResponse)
	wantContentRange := fmt.Sprintf("bytes %d-%d/%d", rangeStart, rangeEnd, len(installerBytes))
	if rangeResponse.StatusCode != http.StatusPartialContent ||
		rangeResponse.Header.Get("Content-Range") != wantContentRange ||
		!bytes.Equal(rangeBody, installerBytes[rangeStart:rangeEnd+1]) {
		t.Fatalf(
			"MDP range response status/range/body = %d/%q/%d bytes, want 206/%q/exact",
			rangeResponse.StatusCode,
			rangeResponse.Header.Get("Content-Range"),
			len(rangeBody),
			wantContentRange,
		)
	}
}

func setupMDPAdmin(t *testing.T, server *testServer) {
	t.Helper()

	var admin struct {
		ID int64 `json:"id"`
	}
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/setup",
		struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Password string `json:"password"`
		}{
			Email:    "mdp-admin@woodstar.test",
			Name:     "MDP Integration Administrator",
			Password: "mdp-integration-admin-password",
		},
		http.StatusCreated,
		&admin,
	)
	if admin.ID <= 0 {
		t.Fatalf("setup admin id = %d, want positive", admin.ID)
	}
}

func createMDPMunkiSecret(t *testing.T, server *testServer, secret string) {
	t.Helper()

	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/agent-secrets",
		struct {
			Agent string `json:"agent"`
			Value string `json:"value"`
		}{Agent: "munki", Value: secret},
		http.StatusCreated,
		nil,
	)
}

func createMDPInstaller(
	t *testing.T,
	server *testServer,
	filename string,
	contents []byte,
) mdpObject {
	t.Helper()

	var target mdpUploadTarget
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/munki/package-installers",
		struct {
			Filename string `json:"filename"`
		}{Filename: filename},
		http.StatusCreated,
		&target,
	)
	if target.ObjectID <= 0 || target.Upload.URL == "" || target.Upload.Method != http.MethodPut ||
		target.Upload.Strategy != "direct-put" {
		t.Fatalf(
			"installer upload target object/method/strategy/has URL = %d/%q/%q/%t, want positive object, PUT, direct-put, and URL",
			target.ObjectID,
			target.Upload.Method,
			target.Upload.Strategy,
			target.Upload.URL != "",
		)
	}

	uploadRequest, err := http.NewRequestWithContext(
		t.Context(),
		target.Upload.Method,
		target.Upload.URL,
		bytes.NewReader(contents),
	)
	if err != nil {
		t.Fatalf("create installer upload request: %v", redactedRequestError(err))
	}
	for name, value := range target.Upload.Headers {
		uploadRequest.Header.Set(name, value)
	}
	uploadResponse, err := verifyingClient(t, server.CACertificate).Do(uploadRequest)
	if err != nil {
		t.Fatalf("upload installer: %v", redactedRequestError(err))
	}
	drainAndClose(t, uploadResponse)
	if uploadResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("installer upload status = %d, want %d", uploadResponse.StatusCode, http.StatusNoContent)
	}

	var object mdpObject
	requestJSON(
		t,
		server.Client,
		http.MethodPut,
		server.BaseURL+"/api/munki/package-installers/"+strconv.FormatInt(target.ObjectID, 10),
		nil,
		http.StatusOK,
		&object,
	)
	if object.ID != target.ObjectID || object.Filename != filename {
		t.Fatalf("finalized installer = %+v, want uploaded object", object)
	}
	return object
}

func createMDPSoftware(t *testing.T, server *testServer) int64 {
	t.Helper()

	var software struct {
		ID int64 `json:"id"`
	}
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/munki/software",
		struct {
			Name        string          `json:"name"`
			DisplayName string          `json:"display_name"`
			Description string          `json:"description"`
			Targets     mdpEmptyTargets `json:"targets"`
		}{
			Name:        "WoodstarMDPIntegration",
			DisplayName: "Woodstar MDP Integration",
			Description: "Compiled distribution point lifecycle fixture.",
			Targets: mdpEmptyTargets{
				Include: []struct{}{},
				Exclude: []struct{}{},
			},
		},
		http.StatusCreated,
		&software,
	)
	if software.ID <= 0 {
		t.Fatalf("created software id = %d, want positive", software.ID)
	}
	return software.ID
}

func createMDPPackage(
	t *testing.T,
	server *testServer,
	softwareID int64,
	installer mdpObject,
) mdpPackage {
	t.Helper()

	var pkg mdpPackage
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
			SoftwareID:        softwareID,
			Version:           "1.0",
			InstallerType:     "pkg",
			InstallerObjectID: installer.ID,
		},
		http.StatusCreated,
		&pkg,
	)
	if pkg.ID <= 0 || pkg.Software.ID != softwareID || pkg.InstallerFile == nil ||
		pkg.InstallerFile.Filename != installer.Filename ||
		pkg.InstallerFile.InstallerItemLocation == "" {
		t.Fatalf("created package = %+v, want finalized installer", pkg)
	}
	return pkg
}

func createMDPDistributionPoint(
	t *testing.T,
	server *testServer,
	workerBaseURL string,
) mdpDistributionPoint {
	t.Helper()

	var point mdpDistributionPoint
	requestJSON(
		t,
		server.Client,
		http.MethodPost,
		server.BaseURL+"/api/munki/distribution-points",
		struct {
			Name          string   `json:"name"`
			Enabled       bool     `json:"enabled"`
			ClientCIDRs   []string `json:"client_cidrs"`
			ClientBaseURL string   `json:"client_base_url"`
		}{
			Name:          "Local integration MDP",
			Enabled:       true,
			ClientCIDRs:   []string{"127.0.0.0/8"},
			ClientBaseURL: workerBaseURL,
		},
		http.StatusCreated,
		&point,
	)
	if point.ID <= 0 || !point.Enabled || point.ClientBaseURL != workerBaseURL {
		t.Fatalf(
			"created distribution point id/enabled/client URL = %d/%t/%q, want enabled loopback point",
			point.ID,
			point.Enabled,
			point.ClientBaseURL,
		)
	}
	return point
}

func waitForMDPCurrent(
	parent context.Context,
	server *testServer,
	pointID int64,
	packageID int64,
	process *serverProcess,
) (mdpDistributionPoint, error) {
	ctx, cancel := context.WithTimeout(parent, mdpLifecycleTimeout)
	defer cancel()

	endpoint := server.BaseURL + "/api/munki/distribution-points/" + strconv.FormatInt(pointID, 10)
	backoff := 10 * time.Millisecond
	for {
		point, lastState, ready := observeMDPState(ctx, server.Client, endpoint, packageID)
		if ready {
			return point, nil
		}

		timer := time.NewTimer(backoff)
		select {
		case <-process.done:
			timer.Stop()
			if process.waitErr == nil {
				return mdpDistributionPoint{}, errors.New("MDP worker exited before mirroring package")
			}
			return mdpDistributionPoint{}, fmt.Errorf(
				"MDP worker exited before mirroring package: %w",
				process.waitErr,
			)
		case <-ctx.Done():
			timer.Stop()
			return mdpDistributionPoint{}, fmt.Errorf("deadline: %w (last state: %s)", ctx.Err(), lastState)
		case <-timer.C:
		}
		backoff = min(backoff*2, 250*time.Millisecond)
	}
}

func observeMDPState(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	packageID int64,
) (mdpDistributionPoint, string, bool) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return mdpDistributionPoint{}, "create distribution point request: " + err.Error(), false
	}
	response, err := client.Do(request)
	if err != nil {
		return mdpDistributionPoint{}, "request detail: " + err.Error(), false
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	closeErr := response.Body.Close()
	switch {
	case readErr != nil:
		return mdpDistributionPoint{}, "read detail: " + readErr.Error(), false
	case closeErr != nil:
		return mdpDistributionPoint{}, "close detail: " + closeErr.Error(), false
	case response.StatusCode != http.StatusOK:
		return mdpDistributionPoint{}, "detail status " + response.Status, false
	}

	var point mdpDistributionPoint
	if err := json.Unmarshal(body, &point); err != nil {
		return mdpDistributionPoint{}, "decode detail: " + err.Error(), false
	}
	state := mdpStateSummary(point, packageID)
	ready := point.Online && mdpPackageIsCurrent(point.Packages, packageID)
	return point, state, ready
}

func mdpPackageIsCurrent(states []mdpPackageState, packageID int64) bool {
	for _, state := range states {
		if state.PackageID == packageID && state.Status == "current" {
			return true
		}
	}
	return false
}

func mdpStateSummary(point mdpDistributionPoint, packageID int64) string {
	status := "missing"
	for _, state := range point.Packages {
		if state.PackageID == packageID {
			status = state.Status
			break
		}
	}
	return fmt.Sprintf("online=%t package=%s", point.Online, status)
}

func requireMDPRedirect(
	t *testing.T,
	response *http.Response,
	workerBaseURL string,
	installerItemLocation string,
) string {
	t.Helper()

	if response.StatusCode != http.StatusFound {
		t.Fatalf("Munki package status = %d, want MDP redirect %d", response.StatusCode, http.StatusFound)
	}
	location := response.Header.Get("Location")
	redirect, err := url.Parse(location)
	if err != nil {
		t.Fatal("parse MDP redirect URL")
	}
	worker, err := url.Parse(workerBaseURL)
	if err != nil {
		t.Fatalf("parse MDP worker URL: %v", err)
	}
	wantPath := "/munki/pkgs/" + installerItemLocation
	hasCapability := strings.TrimSpace(redirect.Query().Get("cap")) != ""
	if redirect.Scheme != worker.Scheme || redirect.Host != worker.Host ||
		redirect.Path != wantPath || !hasCapability {
		t.Fatalf(
			"MDP redirect scheme/host/path/has capability = %q/%q/%q/%t, want %q/%q/%q/true",
			redirect.Scheme,
			redirect.Host,
			redirect.Path,
			hasCapability,
			worker.Scheme,
			worker.Host,
			wantPath,
		)
	}
	return location
}

func redactedRequestError(err error) error {
	var requestErr *url.Error
	if !errors.As(err, &requestErr) {
		return err
	}
	redacted := *requestErr
	requestURL, parseErr := url.Parse(redacted.URL)
	if parseErr != nil {
		redacted.URL = "[redacted]"
		return &redacted
	}
	requestURL.RawQuery = ""
	requestURL.Fragment = ""
	redacted.URL = requestURL.String()
	return &redacted
}
