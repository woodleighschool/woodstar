package integration

import (
	"context"
	"debug/elf"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

const (
	orbitBinaryEnvironment         = "WOODSTAR_ORBIT_BINARY"
	osquerydBinaryEnvironment      = "WOODSTAR_OSQUERYD_BINARY"
	integrationRequiredEnvironment = "WOODSTAR_INTEGRATION_REQUIRED"
	orbitContainerImage            = "debian:bookworm-slim@" +
		"sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818"
	orbitExpectedVersion       = "1.57.0"
	osquerydExpectedVersion    = "5.23.1"
	orbitProviderTimeout       = 5 * time.Second
	orbitContainerStartTimeout = 2 * time.Minute
	orbitEnrollmentTimeout     = 3 * time.Minute
	orbitIterationTimeout      = 5 * time.Second
	orbitCleanupTimeout        = 20 * time.Second
	orbitStopTimeout           = 10 * time.Second
)

var orbitLogCredentialPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)((?:orbit_)?node_key|device_auth_token)(["'=:\s]+)[A-Za-z0-9_-]+`),
	regexp.MustCompile(`\b[A-Za-z0-9_-]{43}\b`),
	regexp.MustCompile(`\b[0-9A-Fa-f]{64}\b`),
	regexp.MustCompile(`\b[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}\b`),
}

type orbitArtifacts struct {
	orbitPath   string
	osqueryPath string
	platform    string
	target      string
}

type orbitTestHost struct {
	Enrollment struct {
		Agent string `json:"agent"`
	} `json:"enrollment"`
	Hardware struct {
		UUID string `json:"uuid"`
	} `json:"hardware"`
	Agents struct {
		Osquery struct {
			Version string `json:"version"`
		} `json:"osquery"`
		Orbit struct {
			Version string `json:"version"`
		} `json:"orbit"`
	} `json:"agents"`
}

type orbitTestHostList struct {
	Items []orbitTestHost `json:"items"`
	Count int             `json:"count"`
}

type orbitPublicResult struct {
	hosts   orbitTestHostList
	summary string
	err     error
}

type orbitContainerStatus struct {
	running bool
	summary string
	err     error
}

func TestOrbit(t *testing.T) {
	artifacts := prepareOrbitArtifacts(t)
	server := startTestServer(t)

	var setupUser struct {
		Email string `json:"email"`
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
			Email:    "admin@woodstar.test",
			Name:     "Integration Administrator",
			Password: "integration-admin-password",
		},
		http.StatusCreated,
		&setupUser,
	)
	if setupUser.Email != "admin@woodstar.test" {
		t.Fatalf("setup email = %q, want admin@woodstar.test", setupUser.Email)
	}

	enrollSecret := randomHex(t, 32)
	server.redact(enrollSecret)
	containerRedactions := []string{enrollSecret}
	var createdSecret struct {
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
		}{Agent: "orbit", Value: enrollSecret},
		http.StatusCreated,
		&createdSecret,
	)
	if createdSecret.Agent != "orbit" {
		t.Fatalf("created agent secret = %q, want orbit", createdSecret.Agent)
	}

	fixtureRoot := t.TempDir()
	secretPath := filepath.Join(fixtureRoot, "secret.txt")
	if err := os.WriteFile(secretPath, []byte(enrollSecret), 0o600); err != nil {
		t.Fatalf("write Orbit enrollment secret fixture: %v", err)
	}
	flagsPath := filepath.Join(fixtureRoot, "osquery.flags")
	flags := "--disable_carver=true\n--carver_disable_function=true\n--logger_min_status=4\n"
	if err := os.WriteFile(flagsPath, []byte(flags), 0o600); err != nil {
		t.Fatalf("write Orbit osquery flags fixture: %v", err)
	}

	serverURL, err := url.Parse(server.BaseURL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	port, err := strconv.Atoi(serverURL.Port())
	if err != nil {
		t.Fatalf("parse test server port %q: %v", serverURL.Port(), err)
	}
	command := orbitContainerCommand(artifacts.target, port)
	lastPublic := "(no public response yet)"

	startCtx, startCancel := context.WithTimeout(t.Context(), orbitContainerStartTimeout)
	container, runErr := testcontainers.Run(
		startCtx,
		orbitContainerImage,
		testcontainers.WithImagePlatform(artifacts.platform),
		testcontainers.WithHostPortAccess(port),
		testcontainers.WithFiles(
			testcontainers.ContainerFile{
				HostFilePath:      artifacts.orbitPath,
				ContainerFilePath: "/usr/local/bin/orbit",
				FileMode:          0o755,
			},
			testcontainers.ContainerFile{
				HostFilePath:      artifacts.osqueryPath,
				ContainerFilePath: "/tmp/woodstar-osqueryd",
				FileMode:          0o755,
			},
			testcontainers.ContainerFile{
				HostFilePath:      server.CACertificatePath,
				ContainerFilePath: "/tmp/woodstar-ca.pem",
				FileMode:          0o644,
			},
			testcontainers.ContainerFile{
				HostFilePath:      secretPath,
				ContainerFilePath: "/tmp/woodstar-secret",
				FileMode:          0o600,
			},
			testcontainers.ContainerFile{
				HostFilePath:      flagsPath,
				ContainerFilePath: "/tmp/woodstar-osquery.flags",
				FileMode:          0o600,
			},
		),
		testcontainers.WithCmd("/bin/sh", "-ec", command),
	)
	if container != nil {
		t.Cleanup(func() {
			failedBeforeCleanup := t.Failed()
			if failedBeforeCleanup {
				t.Logf(
					"Orbit container diagnostics:\n%s",
					orbitFailureDetails(container, server, containerRedactions, lastPublic),
				)
			}
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), orbitCleanupTimeout)
			defer cleanupCancel()
			if err := container.Terminate(cleanupCtx, testcontainers.StopTimeout(orbitStopTimeout)); err != nil {
				t.Errorf("terminate Orbit container: %v", err)
				if !failedBeforeCleanup {
					t.Logf(
						"Orbit container cleanup diagnostics:\n%s",
						orbitFailureDetails(container, server, containerRedactions, lastPublic),
					)
				}
			}
		})
	}
	startCancel()
	if runErr != nil {
		if container == nil {
			t.Fatalf("start Orbit container: %v\nWoodstar server logs (tail):\n%s", runErr, server.logs())
		}
		t.Fatalf(
			"start Orbit container: %v\n%s",
			runErr,
			orbitFailureDetails(container, server, containerRedactions, "(no public response yet)"),
		)
	}

	lastPublic, err = waitForOrbitHost(t.Context(), server.Client, server.BaseURL+"/api/hosts", container)
	if err != nil {
		t.Fatalf(
			"wait for real Orbit lifecycle: %v\n%s",
			err,
			orbitFailureDetails(container, server, containerRedactions, lastPublic),
		)
	}

	stateCtx, stateCancel := context.WithTimeout(t.Context(), orbitProviderTimeout)
	status := readOrbitContainerStatus(stateCtx, container)
	stateCancel()
	if status.err != nil || !status.running {
		t.Fatalf(
			"Orbit container did not remain running after public enrollment: %s\n%s",
			status.summary,
			orbitFailureDetails(container, server, containerRedactions, lastPublic),
		)
	}
}

func prepareOrbitArtifacts(t *testing.T) orbitArtifacts {
	t.Helper()

	required := strings.TrimSpace(os.Getenv(integrationRequiredEnvironment)) == "orbit"
	orbitPath, orbitMachine := readOrbitELF(t, orbitBinaryEnvironment, required)
	osqueryPath, osqueryMachine := readOrbitELF(t, osquerydBinaryEnvironment, required)
	if orbitMachine != osqueryMachine {
		t.Fatalf(
			"%s architecture %s does not match %s architecture %s",
			orbitBinaryEnvironment,
			orbitMachine,
			osquerydBinaryEnvironment,
			osqueryMachine,
		)
	}

	artifacts := orbitArtifacts{orbitPath: orbitPath, osqueryPath: osqueryPath}
	switch orbitMachine {
	case elf.EM_X86_64:
		artifacts.platform = "linux/amd64"
		artifacts.target = "linux"
	case elf.EM_AARCH64:
		artifacts.platform = "linux/arm64"
		artifacts.target = "linux-arm64"
	default:
		t.Fatalf("unsupported Orbit artifact architecture %s", orbitMachine)
	}

	provider, err := testcontainers.ProviderDocker.GetProvider()
	if err != nil {
		orbitPrerequisiteUnavailablef(t, required, "Docker provider: %v", err)
	}
	healthCtx, healthCancel := context.WithTimeout(t.Context(), orbitProviderTimeout)
	healthErr := provider.Health(healthCtx)
	healthCancel()
	closeErr := provider.Close()
	if err := errors.Join(healthErr, closeErr); err != nil {
		orbitPrerequisiteUnavailablef(t, required, "Docker provider health: %v", err)
	}

	return artifacts
}

func readOrbitELF(t *testing.T, environment string, required bool) (string, elf.Machine) {
	t.Helper()

	configuredPath := strings.TrimSpace(os.Getenv(environment))
	if configuredPath == "" {
		orbitPrerequisiteUnavailablef(t, required, "%s is not set", environment)
	}
	absolutePath, err := filepath.Abs(configuredPath)
	if err != nil {
		t.Fatalf("resolve %s path %q: %v", environment, configuredPath, err)
	}
	info, err := os.Stat(absolutePath)
	if errors.Is(err, os.ErrNotExist) {
		orbitPrerequisiteUnavailablef(t, required, "%s path %q does not exist", environment, absolutePath)
	}
	if err != nil {
		t.Fatalf("stat %s path %q: %v", environment, absolutePath, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("%s path %q must be a regular file", environment, absolutePath)
	}
	if info.Mode().Perm()&0o444 == 0 {
		t.Fatalf("%s path %q is not readable", environment, absolutePath)
	}

	binary, err := elf.Open(absolutePath)
	if err != nil {
		t.Fatalf("read %s ELF header from %q: %v", environment, absolutePath, err)
	}
	machine := binary.Machine
	if err := binary.Close(); err != nil {
		t.Fatalf("close %s ELF file %q: %v", environment, absolutePath, err)
	}
	if machine != elf.EM_X86_64 && machine != elf.EM_AARCH64 {
		t.Fatalf("%s path %q has unsupported ELF architecture %s", environment, absolutePath, machine)
	}
	return absolutePath, machine
}

func orbitPrerequisiteUnavailablef(t *testing.T, required bool, format string, args ...any) {
	t.Helper()

	reason := fmt.Sprintf(format, args...)
	if required {
		t.Fatalf("Orbit integration prerequisite unavailable: %s", reason)
	}
	t.Skipf("Orbit integration prerequisite unavailable: %s", reason)
}

func orbitContainerCommand(target string, port int) string {
	return fmt.Sprintf(`
mkdir -p /tmp/woodstar-orbit/bin/osqueryd/%s/stable
chmod 0755 /tmp/woodstar-orbit /tmp/woodstar-orbit/bin /tmp/woodstar-orbit/bin/osqueryd /tmp/woodstar-orbit/bin/osqueryd/%s /tmp/woodstar-orbit/bin/osqueryd/%s/stable
mv /tmp/woodstar-osqueryd /tmp/woodstar-orbit/bin/osqueryd/%s/stable/osqueryd
mv /tmp/woodstar-ca.pem /tmp/woodstar-orbit/woodstar-ca.pem
mv /tmp/woodstar-secret /tmp/woodstar-orbit/secret.txt
mv /tmp/woodstar-osquery.flags /tmp/woodstar-orbit/osquery.flags
chmod 0755 /usr/local/bin/orbit /tmp/woodstar-orbit/bin/osqueryd/%s/stable/osqueryd
chmod 0644 /tmp/woodstar-orbit/woodstar-ca.pem
chmod 0600 /tmp/woodstar-orbit/secret.txt /tmp/woodstar-orbit/osquery.flags
exec /usr/local/bin/orbit \
  --root-dir /tmp/woodstar-orbit \
  --fleet-url https://host.testcontainers.internal:%d \
  --fleet-certificate /tmp/woodstar-orbit/woodstar-ca.pem \
  --enroll-secret-path /tmp/woodstar-orbit/secret.txt \
  --disable-updates \
  --disable-setup-experience \
  --disable-keystore
`, target, target, target, target, target, port)
}

func waitForOrbitHost(
	parent context.Context,
	client *http.Client,
	hostsURL string,
	container testcontainers.Container,
) (string, error) {
	ctx, cancel := context.WithTimeout(parent, orbitEnrollmentTimeout)
	defer cancel()

	lastPublic := "(no public response yet)"
	backoff := 100 * time.Millisecond
	for {
		public, status := pollOrbitHostOnce(ctx, client, hostsURL, container)
		if public.summary != "" {
			lastPublic = public.summary
		}
		if status.err != nil {
			return lastPublic, fmt.Errorf("inspect Orbit container: %w", status.err)
		}
		if !status.running {
			return lastPublic, fmt.Errorf("Orbit container exited before enrollment: %s", status.summary)
		}
		if public.err == nil && orbitHostLifecycleReady(public.hosts) {
			return lastPublic, nil
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return lastPublic, fmt.Errorf("Orbit enrollment deadline: %w", ctx.Err())
		case <-timer.C:
		}
		backoff = min(backoff*2, time.Second)
	}
}

func pollOrbitHostOnce(
	parent context.Context,
	client *http.Client,
	hostsURL string,
	container testcontainers.Container,
) (orbitPublicResult, orbitContainerStatus) {
	ctx, cancel := context.WithTimeout(parent, orbitIterationTimeout)
	defer cancel()

	publicChannel := make(chan orbitPublicResult, 1)
	stateChannel := make(chan orbitContainerStatus, 1)
	go func() { publicChannel <- fetchOrbitHosts(ctx, client, hostsURL) }()
	go func() { stateChannel <- readOrbitContainerStatus(ctx, container) }()

	var public orbitPublicResult
	var status orbitContainerStatus
	for publicChannel != nil || stateChannel != nil {
		select {
		case public = <-publicChannel:
			publicChannel = nil
		case status = <-stateChannel:
			stateChannel = nil
			if status.err != nil || !status.running {
				cancel()
				return public, status
			}
		case <-ctx.Done():
			if publicChannel != nil {
				public.err = ctx.Err()
				public.summary = "public host request did not complete: " + ctx.Err().Error()
			}
			if stateChannel != nil {
				status.err = ctx.Err()
				status.summary = "container state did not complete: " + ctx.Err().Error()
			}
			return public, status
		}
	}
	return public, status
}

func fetchOrbitHosts(ctx context.Context, client *http.Client, hostsURL string) orbitPublicResult {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, hostsURL, nil)
	if err != nil {
		return orbitPublicResult{summary: "create public host request: " + err.Error(), err: err}
	}
	response, err := client.Do(request)
	if err != nil {
		return orbitPublicResult{summary: "send public host request: " + err.Error(), err: err}
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, serverLogTailLimit+1))
	closeErr := response.Body.Close()
	if len(body) > serverLogTailLimit {
		body = body[:serverLogTailLimit]
		readErr = errors.Join(readErr, errors.New("public host response exceeded diagnostic limit"))
	}
	summary := fmt.Sprintf("status=%s body=%s", response.Status, strings.TrimSpace(string(body)))
	if err := errors.Join(readErr, closeErr); err != nil {
		return orbitPublicResult{summary: summary + " response_error=" + err.Error(), err: err}
	}
	if response.StatusCode != http.StatusOK {
		return orbitPublicResult{summary: summary, err: fmt.Errorf("public hosts returned %s", response.Status)}
	}

	var hosts orbitTestHostList
	if err := json.Unmarshal(body, &hosts); err != nil {
		decodeErr := fmt.Errorf("decode public hosts response: %w", err)
		return orbitPublicResult{summary: summary + " decode_error=" + err.Error(), err: decodeErr}
	}
	return orbitPublicResult{hosts: hosts, summary: summary}
}

func orbitHostLifecycleReady(hosts orbitTestHostList) bool {
	if hosts.Count != 1 || len(hosts.Items) != 1 {
		return false
	}
	host := hosts.Items[0]
	return host.Hardware.UUID != "" &&
		host.Enrollment.Agent == "osquery" &&
		host.Agents.Osquery.Version == osquerydExpectedVersion &&
		host.Agents.Orbit.Version == orbitExpectedVersion
}

func readOrbitContainerStatus(ctx context.Context, container testcontainers.Container) orbitContainerStatus {
	state, err := container.State(ctx)
	if err != nil {
		return orbitContainerStatus{summary: "state unavailable: " + err.Error(), err: err}
	}
	if state == nil {
		err := errors.New("container returned nil state")
		return orbitContainerStatus{summary: err.Error(), err: err}
	}
	return orbitContainerStatus{
		running: state.Running,
		summary: fmt.Sprintf(
			"status=%s running=%t exit=%d oom=%t error=%q",
			state.Status,
			state.Running,
			state.ExitCode,
			state.OOMKilled,
			state.Error,
		),
	}
}

func orbitFailureDetails(
	container testcontainers.Container,
	server *testServer,
	redactions []string,
	lastPublic string,
) string {
	stateCtx, stateCancel := context.WithTimeout(context.Background(), orbitProviderTimeout)
	status := readOrbitContainerStatus(stateCtx, container)
	stateCancel()
	return fmt.Sprintf(
		"container: %s\nlast public response: %s\nOrbit container logs (tail):\n%s\nWoodstar server logs (tail):\n%s",
		status.summary,
		lastPublic,
		orbitContainerLogTail(container, redactions),
		server.logs(),
	)
}

func orbitContainerLogTail(container testcontainers.Container, redactions []string) string {
	logCtx, logCancel := context.WithTimeout(context.Background(), orbitProviderTimeout)
	defer logCancel()
	reader, err := container.Logs(logCtx)
	if err != nil {
		return "read container logs: " + err.Error()
	}
	contents, readErr := readOrbitBoundedTail(reader, serverLogTailLimit)
	closeErr := reader.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return "read container logs: " + err.Error()
	}
	logs := string(contents)
	for _, secret := range redactions {
		if secret != "" {
			logs = strings.ReplaceAll(logs, secret, "[REDACTED]")
		}
	}
	for _, pattern := range orbitLogCredentialPatterns {
		logs = pattern.ReplaceAllString(logs, "[REDACTED_CREDENTIAL]")
	}
	if logs == "" {
		return "(no container output)"
	}
	return logs
}

func readOrbitBoundedTail(reader io.Reader, limit int) ([]byte, error) {
	contents := make([]byte, 0, limit)
	chunk := make([]byte, 32<<10)
	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			contents = append(contents, chunk[:n]...)
			if len(contents) > limit {
				trimmed := make([]byte, limit)
				copy(trimmed, contents[len(contents)-limit:])
				contents = trimmed
			}
		}
		if errors.Is(err, io.EOF) {
			return contents, nil
		}
		if err != nil {
			return contents, err
		}
	}
}
