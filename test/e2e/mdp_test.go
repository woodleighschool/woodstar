package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/test/e2e/adminapi"
)

const mdpLifecycleTimeout = 30 * time.Second

func TestMDP(t *testing.T) {
	const (
		installerFilename = "WoodstarMDPIntegration.pkg"
		munkiSecret       = "munki-mdp-integration-secret-0123456789"
	)

	server := startTestServer(t)
	server.redact(munkiSecret)
	provisionMDPAdmin(t, server)
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
		point.Id,
		pkg.Id,
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
		detail.ClientBaseUrl != workerBaseURL {
		t.Fatalf(
			"distribution point detail id/name/enabled/client URL = %d/%q/%t/%q, want enabled local integration MDP",
			detail.Id,
			detail.Name,
			detail.Enabled,
			detail.ClientBaseUrl,
		)
	}

	mirroredPath := filepath.Join(workerDataDir, fmt.Sprintf("%d-%s", pkg.Id, installerFilename))
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

func provisionMDPAdmin(t *testing.T, server *testServer) {
	t.Helper()

	provisionAdmin(
		t,
		server,
		"mdp-admin@woodstar.test",
		"MDP Integration Administrator",
		"mdp-integration-admin-password",
	)
}

func createMDPMunkiSecret(t *testing.T, server *testServer, secret string) {
	t.Helper()

	createAgentSecret(t, server, adminapi.AgentSecretCreateAgentMunki, secret)
}

func createMDPInstaller(
	t *testing.T,
	server *testServer,
	filename string,
	contents []byte,
) adminapi.MunkiObjectView {
	t.Helper()

	created, err := server.Admin.CreateMunkiPackageInstallerWithResponse(
		t.Context(),
		adminapi.MunkiUploadRequest{Filename: filename},
	)
	created = requireAPIResponse(t, "create package installer", http.StatusCreated, created, err)
	upload := directUpload(t, created.JSON201)
	if created.JSON201.ObjectId <= 0 || upload.Url == "" || upload.Method != http.MethodPut ||
		upload.Strategy != "direct-put" {
		t.Fatalf(
			"installer upload target object/method/strategy/has URL = %d/%q/%q/%t, want positive object, PUT, direct-put, and URL",
			created.JSON201.ObjectId,
			upload.Method,
			upload.Strategy,
			upload.Url != "",
		)
	}

	uploadRequest, err := http.NewRequestWithContext(
		t.Context(),
		upload.Method,
		upload.Url,
		bytes.NewReader(contents),
	)
	if err != nil {
		t.Fatalf("create installer upload request: %v", redactedRequestError(err))
	}
	if upload.Headers != nil {
		for name, value := range *upload.Headers {
			uploadRequest.Header.Set(name, value)
		}
	}
	uploadResponse, err := verifyingClient(t, server.CACertificate).Do(uploadRequest)
	if err != nil {
		t.Fatalf("upload installer: %v", redactedRequestError(err))
	}
	drainAndClose(t, uploadResponse)
	if uploadResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("installer upload status = %d, want %d", uploadResponse.StatusCode, http.StatusNoContent)
	}

	finalized, err := server.Admin.FinalizeMunkiPackageInstallerWithResponse(
		t.Context(),
		created.JSON201.ObjectId,
	)
	finalized = requireAPIResponse(t, "finalize package installer", http.StatusOK, finalized, err)
	if finalized.JSON200 == nil || finalized.JSON200.Id != created.JSON201.ObjectId ||
		finalized.JSON200.Filename != filename {
		t.Fatalf("finalized installer = %+v, want uploaded object", finalized.JSON200)
	}
	return *finalized.JSON200
}

func createMDPSoftware(t *testing.T, server *testServer) int64 {
	t.Helper()

	created, err := server.Admin.CreateMunkiSoftwareWithResponse(
		t.Context(),
		adminapi.MunkiCreateMutation{
			Name:        "WoodstarMDPIntegration",
			DisplayName: new("Woodstar MDP Integration"),
			Description: new("Compiled distribution point lifecycle fixture."),
			Targets: adminapi.MunkiTargets{
				Include: []adminapi.MunkiInclude{},
				Exclude: []adminapi.LabelRef{},
			},
		},
	)
	created = requireAPIResponse(t, "create software", http.StatusCreated, created, err)
	if created.JSON201 == nil || created.JSON201.Id <= 0 {
		t.Fatalf("created software = %+v, want positive ID", created.JSON201)
	}
	return created.JSON201.Id
}

func createMDPPackage(
	t *testing.T,
	server *testServer,
	softwareID int64,
	installer adminapi.MunkiObjectView,
) adminapi.MunkiPackage {
	t.Helper()

	created, err := server.Admin.CreateMunkiPackageWithResponse(
		t.Context(),
		adminapi.MunkiPackageCreateMutation{
			SoftwareId:        softwareID,
			Version:           "1.0",
			InstallerType:     new(adminapi.MunkiPackageCreateMutationInstallerType("pkg")),
			InstallerObjectId: new(installer.Id),
		},
	)
	created = requireAPIResponse(t, "create package", http.StatusCreated, created, err)
	if created.JSON201 == nil {
		t.Fatal("create package returned no JSON body")
	}
	pkg := *created.JSON201
	if pkg.Id <= 0 || pkg.Software.Id != softwareID || pkg.InstallerFile == nil ||
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
) adminapi.MunkiRevealedDistributionPoint {
	t.Helper()

	created, err := server.Admin.CreateMunkiDistributionPointWithResponse(
		t.Context(),
		adminapi.MunkiDistributionPointMutation{
			Name:          "Local integration MDP",
			Enabled:       true,
			ClientCidrs:   []string{"127.0.0.0/8"},
			ClientBaseUrl: workerBaseURL,
		},
	)
	created = requireAPIResponse(t, "create distribution point", http.StatusCreated, created, err)
	if created.JSON201 == nil {
		t.Fatal("create distribution point returned no JSON body")
	}
	point := *created.JSON201
	if point.Id <= 0 || !point.Enabled || point.ClientBaseUrl != workerBaseURL {
		t.Fatalf(
			"created distribution point id/enabled/client URL = %d/%t/%q, want enabled loopback point",
			point.Id,
			point.Enabled,
			point.ClientBaseUrl,
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
) (adminapi.MunkiDistributionPointDetail, error) {
	ctx, cancel := context.WithTimeout(parent, mdpLifecycleTimeout)
	defer cancel()

	backoff := 10 * time.Millisecond
	for {
		point, lastState, ready := observeMDPState(ctx, server.Admin, pointID, packageID)
		if ready {
			return point, nil
		}

		timer := time.NewTimer(backoff)
		select {
		case <-process.done:
			timer.Stop()
			if process.waitErr == nil {
				return adminapi.MunkiDistributionPointDetail{}, errors.New("MDP worker exited before mirroring package")
			}
			return adminapi.MunkiDistributionPointDetail{}, fmt.Errorf(
				"MDP worker exited before mirroring package: %w",
				process.waitErr,
			)
		case <-ctx.Done():
			timer.Stop()
			return adminapi.MunkiDistributionPointDetail{}, fmt.Errorf(
				"deadline: %w (last state: %s)",
				ctx.Err(),
				lastState,
			)
		case <-timer.C:
		}
		backoff = min(backoff*2, 250*time.Millisecond)
	}
}

func observeMDPState(
	ctx context.Context,
	client *adminapi.ClientWithResponses,
	pointID int64,
	packageID int64,
) (adminapi.MunkiDistributionPointDetail, string, bool) {
	response, err := client.GetMunkiDistributionPointWithResponse(ctx, pointID)
	if err != nil {
		return adminapi.MunkiDistributionPointDetail{}, "request detail: " + err.Error(), false
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return adminapi.MunkiDistributionPointDetail{}, fmt.Sprintf("detail status %d", response.StatusCode()), false
	}

	point := *response.JSON200
	state := mdpStateSummary(point, packageID)
	ready := point.Online && mdpPackageIsCurrent(point.Packages, packageID)
	return point, state, ready
}

func mdpPackageIsCurrent(states []adminapi.MunkiPackageState, packageID int64) bool {
	for _, state := range states {
		if state.PackageId == packageID && state.Status == "current" {
			return true
		}
	}
	return false
}

func mdpStateSummary(point adminapi.MunkiDistributionPointDetail, packageID int64) string {
	status := "missing"
	for _, state := range point.Packages {
		if state.PackageId == packageID {
			status = string(state.Status)
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
