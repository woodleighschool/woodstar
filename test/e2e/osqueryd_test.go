package e2e

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"

	"github.com/woodleighschool/woodstar/test/e2e/adminapi"
)

const (
	osquerydContainerImage        = "osquery/osquery:5.17.0-ubuntu24.04"
	osquerydExpectedVersion       = "5.17.0"
	osquerydProviderTimeout       = 5 * time.Second
	osquerydContainerStartTimeout = 2 * time.Minute
	osquerydEnrollmentTimeout     = 30 * time.Second
	osquerydCleanupTimeout        = 20 * time.Second
	osquerydStopTimeout           = 10 * time.Second
)

func TestOsqueryd(t *testing.T) {
	requireOsquerydProvider(t)
	server := startTestServer(t)

	provisionAdmin(
		t,
		server,
		"admin@woodstar.test",
		"Integration Administrator",
		"integration-admin-password",
	)

	enrollSecret := randomHex(t, 32)
	server.redact(enrollSecret)
	createdSecret := createAgentSecret(t, server, adminapi.AgentSecretCreateAgentOrbit, enrollSecret)
	if createdSecret.Agent != "orbit" {
		t.Fatalf("created agent secret = %q, want orbit", createdSecret.Agent)
	}

	flagsPath := filepath.Join(t.TempDir(), "osquery.flags")
	if err := os.WriteFile(flagsPath, []byte(osquerydFlags()), 0o600); err != nil {
		t.Fatalf("write osqueryd flags: %v", err)
	}

	serverURL, err := url.Parse(server.BaseURL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	port, err := strconv.Atoi(serverURL.Port())
	if err != nil {
		t.Fatalf("parse test server port %q: %v", serverURL.Port(), err)
	}

	startCtx, startCancel := context.WithTimeout(t.Context(), osquerydContainerStartTimeout)
	container, runErr := testcontainers.Run(
		startCtx,
		osquerydContainerImage,
		testcontainers.WithImagePlatform("linux/amd64"),
		testcontainers.WithHostPortAccess(port),
		testcontainers.WithEnv(map[string]string{"ENROLL_SECRET": enrollSecret}),
		testcontainers.WithFiles(
			testcontainers.ContainerFile{
				HostFilePath:      server.CACertificatePath,
				ContainerFilePath: "/etc/osquery/woodstar-ca.pem",
				FileMode:          0o644,
			},
			testcontainers.ContainerFile{
				HostFilePath:      flagsPath,
				ContainerFilePath: "/etc/osquery/woodstar.flags",
				FileMode:          0o600,
			},
		),
		testcontainers.WithCmd(
			"osqueryd",
			"--flagfile=/etc/osquery/woodstar.flags",
			fmt.Sprintf("--tls_hostname=host.testcontainers.internal:%d", port),
		),
	)
	startCancel()
	if runErr != nil {
		t.Fatalf("start osqueryd container: %v\nWoodstar server logs (tail):\n%s", runErr, server.logs())
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), osquerydCleanupTimeout)
		defer cleanupCancel()
		if err := container.Terminate(cleanupCtx, testcontainers.StopTimeout(osquerydStopTimeout)); err != nil {
			t.Errorf("terminate osqueryd container: %v", err)
		}
	})

	waitForOsquerydHost(t, server, container)
}

func requireOsquerydProvider(t *testing.T) {
	t.Helper()

	if os.Getenv("CI") == "" {
		testcontainers.SkipIfProviderIsNotHealthy(t)
	}
}

func osquerydFlags() string {
	return `--force=true
--host_identifier=hostname
--tls_server_certs=/etc/osquery/woodstar-ca.pem
--enroll_secret_env=ENROLL_SECRET
--enroll_tls_endpoint=/api/v1/osquery/enroll
--config_plugin=tls
--config_tls_endpoint=/api/v1/osquery/config
--config_refresh=5
--disable_distributed=false
--distributed_plugin=tls
--distributed_interval=5
--distributed_tls_max_attempts=3
--distributed_tls_read_endpoint=/api/v1/osquery/distributed/read
--distributed_tls_write_endpoint=/api/v1/osquery/distributed/write
--logger_plugin=tls
--logger_tls_endpoint=/api/v1/osquery/log
--logger_tls_period=5
--disable_carver=true
--carver_disable_function=true
--logger_min_status=4
`
}

func waitForOsquerydHost(t *testing.T, server *testServer, container testcontainers.Container) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), osquerydEnrollmentTimeout)
	defer cancel()
	lastResponse := "(no public response yet)"

	for {
		hosts, summary, err := fetchOsquerydHosts(ctx, server.Admin)
		if summary != "" {
			lastResponse = summary
		}
		ready := err == nil && osquerydHostReady(hosts)
		if ctx.Err() != nil {
			t.Fatalf(
				"wait for osqueryd enrollment: %v\nlast public response: %s\nWoodstar server logs (tail):\n%s",
				ctx.Err(),
				lastResponse,
				server.logs(),
			)
		}

		stateCtx, stateCancel := context.WithTimeout(ctx, osquerydProviderTimeout)
		state, stateErr := container.State(stateCtx)
		stateCancel()
		if stateErr != nil {
			t.Fatalf(
				"inspect osqueryd container: %v\nlast public response: %s\nWoodstar server logs (tail):\n%s",
				stateErr,
				lastResponse,
				server.logs(),
			)
		}
		if state == nil || !state.Running {
			t.Fatalf(
				"osqueryd exited before enrollment\nlast public response: %s\nWoodstar server logs (tail):\n%s",
				lastResponse,
				server.logs(),
			)
		}
		if ready {
			return
		}

		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			t.Fatalf(
				"wait for osqueryd enrollment: %v\nlast public response: %s\nWoodstar server logs (tail):\n%s",
				ctx.Err(),
				lastResponse,
				server.logs(),
			)
		case <-timer.C:
		}
	}
}

func fetchOsquerydHosts(
	ctx context.Context,
	client *adminapi.ClientWithResponses,
) (adminapi.PageHost, string, error) {
	response, err := client.ListHostsWithResponse(ctx, nil)
	if err != nil {
		return adminapi.PageHost{}, "", err
	}
	if response == nil {
		return adminapi.PageHost{}, "", errors.New("list hosts returned no response")
	}
	if response.JSON200 == nil {
		return adminapi.PageHost{}, "status=" + response.Status(), fmt.Errorf(
			"public hosts returned %s: %s",
			response.Status(),
			response.Body,
		)
	}
	hosts := *response.JSON200
	return hosts, fmt.Sprintf("status=%s count=%d", response.Status(), hosts.Count), nil
}

func osquerydHostReady(hosts adminapi.PageHost) bool {
	if hosts.Count != 1 || len(hosts.Items) != 1 {
		return false
	}
	host := hosts.Items[0]
	return host.Hardware.Uuid != "" &&
		host.Enrollment.Agent == "osquery" &&
		host.Agents.Osquery.Version == osquerydExpectedVersion
}
