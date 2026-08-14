//go:build e2e

package e2e

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	osquerydReportTimeout         = 45 * time.Second
	osquerydCleanupTimeout        = 20 * time.Second
	osquerydStopTimeout           = 10 * time.Second
)

func TestOsqueryd(t *testing.T) {
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
	report := createOsquerydErrorReport(t, server)
	waitForOsquerydReportError(t, server, report)
}

func createOsquerydErrorReport(t *testing.T, server *testServer) adminapi.OsqueryReport {
	t.Helper()

	labelsResponse, err := server.Admin.ListLabelsWithResponse(
		t.Context(),
		&adminapi.ListLabelsParams{LabelType: new(adminapi.ListLabelsParamsLabelType("builtin"))},
	)
	labelsResponse = requireAPIResponse(t, "list report target labels", http.StatusOK, labelsResponse, err)
	if labelsResponse.JSON200 == nil {
		t.Fatal("list report target labels returned no JSON body")
	}
	var allHostsLabelID int64
	for _, label := range labelsResponse.JSON200.Items {
		if label.BuiltinKey != nil && *label.BuiltinKey == "all-hosts" {
			allHostsLabelID = label.Id
			break
		}
	}
	if allHostsLabelID == 0 {
		t.Fatal("all-hosts label not found")
	}

	response, err := server.Admin.CreateOsqueryReportWithResponse(
		t.Context(),
		adminapi.OsqueryReportMutation{
			Name:             "Broken scheduled report",
			Query:            "SELECT * FROM woodstar_intentional_missing_table;",
			ScheduleInterval: new(int32(1)),
			Targets: adminapi.OsqueryReportTargets{
				Include: []adminapi.LabelRef{{LabelId: allHostsLabelID}},
				Exclude: []adminapi.LabelRef{},
			},
		},
	)
	response = requireAPIResponse(t, "create broken osquery report", http.StatusCreated, response, err)
	if response.JSON201 == nil {
		t.Fatal("create broken osquery report returned no JSON body")
	}
	return *response.JSON201
}

func waitForOsquerydReportError(
	t *testing.T,
	server *testServer,
	report adminapi.OsqueryReport,
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), osquerydReportTimeout)
	defer cancel()
	lastResponse := "(no public response yet)"

	for {
		response, err := server.Admin.ListOsqueryReportSnapshotsWithResponse(ctx, report.Id, nil)
		if err != nil {
			lastResponse = err.Error()
		} else if response != nil {
			lastResponse = fmt.Sprintf("status=%s body=%s", response.Status(), response.Body)
			snapshot, ready := osquerydReportErrorSnapshot(response)
			if ready {
				if snapshot.Error == nil ||
					!strings.Contains(*snapshot.Error, "woodstar_intentional_missing_table") ||
					snapshot.ReportedAt == nil ||
					len(snapshot.Rows) != 0 || snapshot.ResultRowCount != 0 {
					t.Fatalf("errored real-osquery report snapshot = %+v", snapshot)
				}
				return
			}
		}

		if ctx.Err() != nil {
			t.Fatalf(
				"wait for real-osquery report error: %v\nlast public response: %s\nWoodstar server logs (tail):\n%s",
				ctx.Err(),
				lastResponse,
				server.logs(),
			)
		}
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

func osquerydReportErrorSnapshot(
	response *adminapi.ListOsqueryReportSnapshotsResponse,
) (adminapi.OsqueryReportSnapshot, bool) {
	if response.JSON200 == nil || len(response.JSON200.Items) != 1 {
		return adminapi.OsqueryReportSnapshot{}, false
	}
	snapshot := response.JSON200.Items[0]
	return snapshot, snapshot.Status == adminapi.OsqueryReportSnapshotStatusError
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
--logger_min_status=2
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
