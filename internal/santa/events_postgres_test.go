//go:build postgres

package santa_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

type uploadedEventFixture struct {
	db                *database.DB
	eventStore        *santaevents.Store
	allowEvent        santaevents.ExecutionEvent
	blockEvent        santaevents.ExecutionEvent
	signingTime       time.Time
	secureSigningTime time.Time
	bundleHash        string
}

func newUploadedEventFixture(t *testing.T) uploadedEventFixture {
	t.Helper()

	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	store := santa.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewSyncService(santa.Dependencies{
		HostStore:      store,
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
		Heartbeats:     heartbeats.NewStore(db),
	})

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware: hosts.HostHardware{
			UUID:   "santa-events-host",
			Serial: "SANTAEVENTS",
		},
		OrbitNodeKey: "santa-events-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	occurredAt := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)
	fixture := uploadedEventFixture{
		db:                db,
		eventStore:        eventStore,
		signingTime:       time.Date(2026, 5, 22, 8, 15, 0, 0, time.UTC),
		secureSigningTime: time.Date(2026, 5, 22, 8, 16, 0, 0, time.UTC),
		bundleHash:        strings.Repeat("c", 64),
	}
	_, err = service.EventUpload(ctx, "santa-events-host", heartbeats.Contact{}, santa.EventUploadRequest{
		Events: []santaevents.ExecutionEventInput{
			{
				FileSHA256:      "sha256-a",
				FilePath:        "/Applications/Example.app/Contents/MacOS/Example",
				FileName:        "Example",
				ExecutingUser:   "alice",
				OccurredAt:      occurredAt,
				LoggedInUsers:   []string{" bo\x00b ", "alice", "bob", "\x00"},
				CurrentSessions: []string{"con\x00sole", "ssh\x00", "console"},
				Decision:        santaevents.ExecutionDecisionBlockBinary,
				BundleID:        "com.example.old",
				BundlePath:      "/Applications/Example.app",
				SigningID:       "TEAMID:com.example.old",
				TeamID:          "TEAMID",
				CDHash:          "old-cdhash",
				SigningChain:    santaTestSigningChain(),
			},
			{
				FileSHA256:              "sha256-a",
				FilePath:                "/Applications/Example.app/Contents/MacOS/Example",
				FileName:                "Example Renamed",
				ExecutingUser:           "bob",
				OccurredAt:              occurredAt.Add(time.Second),
				Decision:                santaevents.ExecutionDecisionAllowBinary,
				BundleID:                "com.example.new",
				BundlePath:              "/Applications/Example.app",
				BundleExecutableRelPath: "Contents/MacOS/Example",
				BundleName:              "Example",
				BundleVersion:           "2.0.0",
				BundleVersionString:     "2.0.0 (42)",
				BundleHash:              fixture.bundleHash,
				BundleHashMillis:        23,
				BundleBinaryCount:       1,
				SigningID:               "TEAMID:com.example.new",
				TeamID:                  "TEAMID",
				CDHash:                  "new-cdhash",
				CodesigningFlags:        570425345,
				SigningStatus:           santaevents.SigningStatusProduction,
				SecureSigningTime:       fixture.secureSigningTime,
				SigningTime:             fixture.signingTime,
				Entitlements: []byte(
					`{"application-identifier":"TEAMID.com.example.new","com.apple.security.cs.allow-jit":true}`,
				),
			},
		},
	})
	if err != nil {
		t.Fatalf("event upload: %v", err)
	}

	items, _, err := eventStore.ListEvents(ctx, santaevents.ExecutionEventListParams{
		EventListParams: santaevents.EventListParams{HostID: host.ID},
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("events = %+v, want two execution events", items)
	}
	eventsByDecision := make(map[santaevents.ExecutionDecision]santaevents.ExecutionEvent, len(items))
	for _, event := range items {
		eventsByDecision[event.Decision] = event
	}
	fixture.allowEvent = eventsByDecision[santaevents.ExecutionDecisionAllowBinary]
	fixture.blockEvent = eventsByDecision[santaevents.ExecutionDecisionBlockBinary]
	if fixture.allowEvent.ID == 0 || fixture.blockEvent.ID == 0 {
		t.Fatalf("events = %+v, want allow_binary and block_binary", items)
	}
	return fixture
}

func TestEventUploadProjectsExecutableDetails(t *testing.T) {
	fixture := newUploadedEventFixture(t)

	t.Run("replaces executable metadata", func(t *testing.T) {
		assertEventUploadReplacesExecutableMetadata(t, fixture)
	})
	t.Run("normalizes joined lists", func(t *testing.T) {
		assertEventUploadNormalizesJoinedLists(t, fixture)
	})
	t.Run("persists signing chain", func(t *testing.T) {
		assertEventUploadPersistsSigningChain(t, fixture)
	})
}

func assertEventUploadReplacesExecutableMetadata(t *testing.T, fixture uploadedEventFixture) {
	t.Helper()

	executable := fixture.allowEvent.Executable

	if executable.FileName != "Example Renamed" ||
		executable.BundleID != "com.example.new" ||
		executable.BundleExecutableRelPath != "Contents/MacOS/Example" ||
		executable.BundleName != "Example" ||
		executable.BundleVersion != "2.0.0" ||
		executable.BundleVersionString != "2.0.0 (42)" ||
		executable.BundleHash != fixture.bundleHash ||
		executable.BundleHashMillis != 23 ||
		executable.BundleBinaryCount != 1 ||
		executable.SigningID != "TEAMID:com.example.new" ||
		executable.CDHash != "new-cdhash" ||
		executable.CodesigningFlags != 570425345 ||
		executable.SigningStatus != santaevents.SigningStatusProduction {
		t.Fatalf("executable metadata was not updated: %+v", executable)
	}
	if executable.SecureSigningTime == nil || !executable.SecureSigningTime.Equal(fixture.secureSigningTime) {
		t.Fatalf("secure signing time = %v, want %v", executable.SecureSigningTime, fixture.secureSigningTime)
	}
	if executable.SigningTime == nil || !executable.SigningTime.Equal(fixture.signingTime) {
		t.Fatalf("signing time = %v, want %v", executable.SigningTime, fixture.signingTime)
	}
	var entitlements map[string]any
	if err := json.Unmarshal(executable.Entitlements, &entitlements); err != nil {
		t.Fatalf("entitlements JSON: %v", err)
	}
	if got := entitlements["application-identifier"]; got != "TEAMID.com.example.new" {
		t.Fatalf("application identifier entitlement = %v, want TEAMID.com.example.new", got)
	}
	if got := entitlements["com.apple.security.cs.allow-jit"]; got != true {
		t.Fatalf("allow-jit entitlement = %v, want true", got)
	}
}

func assertEventUploadNormalizesJoinedLists(t *testing.T, fixture uploadedEventFixture) {
	t.Helper()

	if !slices.Equal(fixture.blockEvent.LoggedInUsers, []string{"bob", "alice"}) {
		t.Fatalf("logged_in_users = %v, want client order", fixture.blockEvent.LoggedInUsers)
	}
	if !slices.Equal(fixture.blockEvent.CurrentSessions, []string{"console", "ssh"}) {
		t.Fatalf("current_sessions = %v, want client order", fixture.blockEvent.CurrentSessions)
	}
	if len(fixture.allowEvent.LoggedInUsers) != 0 {
		t.Fatalf("omitted logged_in_users = %v, want empty array", fixture.allowEvent.LoggedInUsers)
	}
	if len(fixture.allowEvent.CurrentSessions) != 0 {
		t.Fatalf("omitted current_sessions = %v, want empty array", fixture.allowEvent.CurrentSessions)
	}
}

func assertEventUploadPersistsSigningChain(t *testing.T, fixture uploadedEventFixture) {
	t.Helper()

	ctx := t.Context()

	var chainCount int
	if err := fixture.db.Pool().
		QueryRow(ctx, `SELECT count(*) FROM santa_signing_chains`).
		Scan(&chainCount); err != nil {
		t.Fatalf("count signing chains: %v", err)
	}
	if chainCount != 1 {
		t.Fatalf("signing chain count = %d, want 1", chainCount)
	}
	var certificateCount int
	if err := fixture.db.Pool().
		QueryRow(ctx, `SELECT count(*) FROM santa_certificates`).
		Scan(&certificateCount); err != nil {
		t.Fatalf("count certificates: %v", err)
	}
	if certificateCount != 2 {
		t.Fatalf("certificate count = %d, want 2", certificateCount)
	}
	var chainEntryCount int
	if err := fixture.db.Pool().
		QueryRow(ctx, `SELECT count(*) FROM santa_signing_chain_entries`).
		Scan(&chainEntryCount); err != nil {
		t.Fatalf("count signing chain entries: %v", err)
	}
	if chainEntryCount != 2 {
		t.Fatalf("signing chain entry count = %d, want 2", chainEntryCount)
	}
	var linkCount int
	if err := fixture.db.Pool().
		QueryRow(ctx, `SELECT count(*) FROM santa_executable_signing_chains`).
		Scan(&linkCount); err != nil {
		t.Fatalf("count signing chain links: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("signing chain link count = %d, want 1", linkCount)
	}

	detail, err := fixture.eventStore.GetExecutionEvent(ctx, fixture.blockEvent.ID)
	if err != nil {
		t.Fatalf("get execution event: %v", err)
	}
	if len(detail.Executable.SigningChain) != 2 ||
		detail.Executable.SigningChain[0].CommonName != "Leaf" ||
		detail.Executable.SigningChain[1].SHA256 != "root-sha" {
		t.Fatalf("detail signing chain = %+v, want full chain", detail.Executable.SigningChain)
	}
}

func TestEventUploadRequestsAndCollectsBundleBinaries(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewSyncService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
		Heartbeats:     heartbeats.NewStore(db),
	})

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-bundle-events-host"},
		OrbitNodeKey: "santa-bundle-events-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	bundleHash := strings.Repeat("b", 64)
	firstResponse, err := service.EventUpload(ctx, "santa-bundle-events-host", heartbeats.Contact{}, santa.EventUploadRequest{
		Events: []santaevents.ExecutionEventInput{{
			FileSHA256:              strings.Repeat("1", 64),
			FilePath:                "/Applications/Bundle.app/Contents/MacOS/Bundle",
			FileName:                "Bundle",
			OccurredAt:              time.Date(2026, 5, 24, 11, 0, 0, 0, time.UTC),
			Decision:                santaevents.ExecutionDecisionAllowBinary,
			BundleID:                "com.example.bundle",
			BundlePath:              "/Applications/Bundle.app",
			BundleExecutableRelPath: "Contents/MacOS/Bundle",
			BundleName:              "Bundle",
			BundleVersion:           "1.2.3",
			BundleVersionString:     "1.2.3 (45)",
			BundleHash:              bundleHash,
			BundleHashMillis:        15,
			BundleBinaryCount:       2,
		}},
	})
	if err != nil {
		t.Fatalf("first event upload: %v", err)
	}
	if !slices.Equal(firstResponse.BundleBinaryRequests, []string{bundleHash}) {
		t.Fatalf("bundle binary requests = %v, want [%s]", firstResponse.BundleBinaryRequests, bundleHash)
	}

	secondResponse, err := service.EventUpload(ctx, "santa-bundle-events-host", heartbeats.Contact{}, santa.EventUploadRequest{
		Events: []santaevents.ExecutionEventInput{{
			FileSHA256:        strings.Repeat("2", 64),
			FileName:          "Bundle Helper",
			Decision:          santaevents.ExecutionDecisionBundleBinary,
			BundleID:          "com.example.bundle",
			BundlePath:        "/Applications/Bundle.app",
			BundleName:        "Bundle",
			BundleVersion:     "1.2.3",
			BundleHash:        bundleHash,
			BundleBinaryCount: 2,
		}},
	})
	if err != nil {
		t.Fatalf("bundle binary upload: %v", err)
	}
	if len(secondResponse.BundleBinaryRequests) != 0 {
		t.Fatalf("second bundle binary requests = %v, want none", secondResponse.BundleBinaryRequests)
	}

	var eventCount int
	if err := db.Pool().
		QueryRow(ctx, `SELECT count(*) FROM santa_execution_events WHERE host_id = $1`, host.ID).
		Scan(&eventCount); err != nil {
		t.Fatalf("count execution events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("execution event count = %d, want only the real execution row", eventCount)
	}

	var binaryCount int
	var collectedCount int
	var uploadedAt *time.Time
	err = db.Pool().QueryRow(ctx, `
		SELECT b.binary_count, count(be.executable_id)::integer, b.uploaded_at
		FROM santa_bundles b
		LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
		WHERE b.sha256 = $1
		GROUP BY b.id
	`, bundleHash).Scan(&binaryCount, &collectedCount, &uploadedAt)
	if err != nil {
		t.Fatalf("get bundle: %v", err)
	}
	if binaryCount != 2 || collectedCount != 2 || uploadedAt == nil {
		t.Fatalf("bundle count/upload = %d/%d/%v, want complete", binaryCount, collectedCount, uploadedAt)
	}
}

func TestEventUploadDerivesBundleCompletionFromFinalBatchState(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewSyncService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
		Heartbeats:     heartbeats.NewStore(db),
	})

	if _, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-final-bundle-state-host"},
		OrbitNodeKey: "santa-final-bundle-state-orbit",
	}); err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	bundleHash := strings.Repeat("d", 64)
	response, err := service.EventUpload(ctx, "santa-final-bundle-state-host", heartbeats.Contact{}, santa.EventUploadRequest{
		Events: []santaevents.ExecutionEventInput{
			{
				FileSHA256:        strings.Repeat("4", 64),
				OccurredAt:        time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
				Decision:          santaevents.ExecutionDecisionAllowBinary,
				BundleHash:        bundleHash,
				BundleBinaryCount: 1,
			},
			{
				FileSHA256:        strings.Repeat("4", 64),
				OccurredAt:        time.Date(2026, 7, 27, 12, 1, 0, 0, time.UTC),
				Decision:          santaevents.ExecutionDecisionAllowBinary,
				BundleHash:        bundleHash,
				BundleBinaryCount: 2,
			},
		},
	})
	if err != nil {
		t.Fatalf("event upload: %v", err)
	}
	if !slices.Equal(response.BundleBinaryRequests, []string{bundleHash}) {
		t.Fatalf("bundle binary requests = %v, want final incomplete bundle", response.BundleBinaryRequests)
	}

	var binaryCount int
	var collectedCount int
	var uploadedAt *time.Time
	if err := db.Pool().QueryRow(ctx, `
SELECT b.binary_count, count(be.executable_id)::integer, b.uploaded_at
FROM santa_bundles b
LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
WHERE b.sha256 = $1
GROUP BY b.id`, bundleHash).Scan(&binaryCount, &collectedCount, &uploadedAt); err != nil {
		t.Fatalf("get bundle: %v", err)
	}
	if binaryCount != 2 || collectedCount != 1 || uploadedAt != nil {
		t.Fatalf(
			"bundle count/upload = %d/%d/%v, want final incomplete state",
			binaryCount,
			collectedCount,
			uploadedAt,
		)
	}
}

func TestEventUploadReopensBundleWhenExpectedCountIncreases(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewSyncService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
		Heartbeats:     heartbeats.NewStore(db),
	})

	if _, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-reopened-bundle-host"},
		OrbitNodeKey: "santa-reopened-bundle-orbit",
	}); err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	bundleHash := strings.Repeat("e", 64)
	event := santaevents.ExecutionEventInput{
		FileSHA256:        strings.Repeat("5", 64),
		OccurredAt:        time.Date(2026, 7, 27, 13, 0, 0, 0, time.UTC),
		Decision:          santaevents.ExecutionDecisionAllowBinary,
		BundleHash:        bundleHash,
		BundleBinaryCount: 1,
	}
	first, err := service.EventUpload(ctx, "santa-reopened-bundle-host", heartbeats.Contact{}, santa.EventUploadRequest{
		Events: []santaevents.ExecutionEventInput{event},
	})
	if err != nil {
		t.Fatalf("complete bundle upload: %v", err)
	}
	if len(first.BundleBinaryRequests) != 0 {
		t.Fatalf("complete bundle requests = %v, want none", first.BundleBinaryRequests)
	}

	event.OccurredAt = event.OccurredAt.Add(time.Minute)
	event.BundleBinaryCount = 2
	second, err := service.EventUpload(ctx, "santa-reopened-bundle-host", heartbeats.Contact{}, santa.EventUploadRequest{
		Events: []santaevents.ExecutionEventInput{event},
	})
	if err != nil {
		t.Fatalf("increased bundle upload: %v", err)
	}
	if !slices.Equal(second.BundleBinaryRequests, []string{bundleHash}) {
		t.Fatalf("increased bundle requests = %v, want reopened bundle", second.BundleBinaryRequests)
	}

	var uploadedAt *time.Time
	if err := db.Pool().
		QueryRow(ctx, `SELECT uploaded_at FROM santa_bundles WHERE sha256 = $1`, bundleHash).
		Scan(&uploadedAt); err != nil {
		t.Fatalf("get bundle completion: %v", err)
	}
	if uploadedAt != nil {
		t.Fatalf("uploaded_at = %v, want incomplete bundle", uploadedAt)
	}
}

func TestEventUploadIngestsFileAccessEvents(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewSyncService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
		Heartbeats:     heartbeats.NewStore(db),
	})

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-file-access-host"},
		Hostname:     "file-access.example.test",
		OrbitNodeKey: "santa-file-access-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	occurredAt := time.Date(2026, 5, 24, 9, 15, 0, 0, time.UTC)
	_, err = service.EventUpload(ctx, "santa-file-access-host", heartbeats.Contact{}, santa.EventUploadRequest{
		FileAccessEvents: []santaevents.FileAccessEventInput{{
			RuleVersion: "v7",
			RuleName:    "Protect Payroll",
			Target:      "/Users/alice/Payroll.csv",
			Decision:    santaevents.FileAccessDecisionDeniedInvalidSignature,
			OccurredAt:  occurredAt,
			ProcessChain: []santaevents.ProcessInput{
				{
					PID:          100,
					FilePath:     "/Applications/Sketchy.app/Contents/MacOS/Sketchy",
					FileSHA256:   "process-sha",
					SigningID:    "EVILTEAM:sketchy",
					TeamID:       "EVILTEAM",
					CDHash:       "process-cdhash",
					SigningChain: santaTestSigningChain(),
				},
				{PID: 1, FilePath: "/sbin/launchd", FileSHA256: "launchd-sha", SigningID: "platform:launchd"},
			},
		}},
	})
	if err != nil {
		t.Fatalf("event upload: %v", err)
	}

	items, count, err := eventStore.ListFileAccessEvents(ctx, santaevents.FileAccessEventListParams{
		EventListParams: santaevents.EventListParams{HostID: host.ID},
	})
	if err != nil {
		t.Fatalf("list file access events: %v", err)
	}
	if count != 1 || len(items) != 1 {
		t.Fatalf("file access events = %+v count=%d, want one", items, count)
	}
	row := items[0]
	if row.Host.DisplayName != host.DisplayName ||
		row.RuleName != "Protect Payroll" ||
		row.PrimaryProcess.FileName != "Sketchy" ||
		row.Decision != santaevents.FileAccessDecisionDeniedInvalidSignature {
		t.Fatalf("file access event row = %+v", row)
	}

	detail, err := eventStore.GetFileAccessEvent(ctx, row.ID)
	if err != nil {
		t.Fatalf("get file access event: %v", err)
	}
	if !detail.OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred_at = %v, want %v", detail.OccurredAt, occurredAt)
	}
	if len(detail.ProcessChain) != 2 ||
		detail.ProcessChain[0].SigningChain[0].CommonName != "Leaf" ||
		detail.ProcessChain[1].FileName != "launchd" {
		t.Fatalf("process chain = %+v, want persisted chain details", detail.ProcessChain)
	}
	var primarySHA256 string
	var primaryPath string
	var primarySigningID string
	var primaryTeamID string
	var primaryCDHash string
	var primaryPID int
	if err := db.Pool().QueryRow(ctx, `
		SELECT
			primary_process_sha256,
			primary_process_path,
			primary_process_signing_id,
			primary_process_team_id,
			primary_process_cdhash,
			primary_process_pid
		FROM santa_file_access_events
		WHERE id = $1
	`, row.ID).Scan(
		&primarySHA256,
		&primaryPath,
		&primarySigningID,
		&primaryTeamID,
		&primaryCDHash,
		&primaryPID,
	); err != nil {
		t.Fatalf("get primary process columns: %v", err)
	}
	if primarySHA256 != "process-sha" ||
		primaryPath != "/Applications/Sketchy.app/Contents/MacOS/Sketchy" ||
		primarySigningID != "EVILTEAM:sketchy" ||
		primaryTeamID != "EVILTEAM" ||
		primaryCDHash != "process-cdhash" ||
		primaryPID != 100 {
		t.Fatalf(
			"primary process columns = %q %q %q %q %q %d",
			primarySHA256,
			primaryPath,
			primarySigningID,
			primaryTeamID,
			primaryCDHash,
			primaryPID,
		)
	}
}

func TestEventListCursorFiltersAndRetention(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	store := santa.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewSyncService(santa.Dependencies{
		HostStore:      store,
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
		Heartbeats:     heartbeats.NewStore(db),
	})

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-event-list-host"},
		OrbitNodeKey: "santa-event-list-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	base := time.Date(2026, 5, 23, 13, 0, 0, 0, time.UTC)
	for i, decision := range []santaevents.ExecutionDecision{
		santaevents.ExecutionDecisionBlockBinary,
		santaevents.ExecutionDecisionAllowBinary,
		santaevents.ExecutionDecisionBlockCertificate,
	} {
		_, err := service.EventUpload(ctx, "santa-event-list-host", heartbeats.Contact{}, santa.EventUploadRequest{
			Events: []santaevents.ExecutionEventInput{{
				FileSHA256: string(rune('a' + i)),
				FileName:   string(rune('A' + i)),
				OccurredAt: base.Add(time.Duration(i) * time.Minute),
				Decision:   decision,
			}},
		})
		if err != nil {
			t.Fatalf("event upload %d: %v", i, err)
		}
	}

	firstPage, count, err := eventStore.ListEvents(
		ctx,
		santaevents.ExecutionEventListParams{
			EventListParams: santaevents.EventListParams{
				HostID:     host.ID,
				ListParams: listing.Params{PageSize: 2},
			},
		},
	)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(firstPage) != 2 || count != 3 {
		t.Fatalf("first page = %+v count=%d, want two items and count 3", firstPage, count)
	}
	secondPage, _, err := eventStore.ListEvents(
		ctx,
		santaevents.ExecutionEventListParams{
			EventListParams: santaevents.EventListParams{
				HostID:     host.ID,
				ListParams: listing.Params{PageSize: 2, PageIndex: 1},
			},
		},
	)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(secondPage) != 1 || secondPage[0].Decision != santaevents.ExecutionDecisionBlockBinary {
		t.Fatalf("second page = %+v, want oldest blocked binary event", secondPage)
	}

	blocked, _, err := eventStore.ListEvents(
		ctx,
		santaevents.ExecutionEventListParams{
			EventListParams: santaevents.EventListParams{HostID: host.ID},
			Decisions:       []santaevents.DecisionFilter{santaevents.DecisionFilterBlocked},
		},
	)
	if err != nil {
		t.Fatalf("list blocked events: %v", err)
	}
	if len(blocked) != 2 {
		t.Fatalf("blocked events = %+v, want two", blocked)
	}

	allowedBinary, _, err := eventStore.ListEvents(
		ctx,
		santaevents.ExecutionEventListParams{
			EventListParams: santaevents.EventListParams{
				ListParams: listing.Params{Q: "B"},
			},
			Decisions: []santaevents.DecisionFilter{santaevents.DecisionFilterAllowed, "block_certificate"},
		},
	)
	if err != nil {
		t.Fatalf("list searched decision events: %v", err)
	}
	if len(allowedBinary) != 2 {
		t.Fatalf("searched decision events = %+v, want allow binary and block certificate", allowedBinary)
	}

	deleted, err := eventStore.SweepEventsBefore(ctx, base.Add(90*time.Second))
	if err != nil {
		t.Fatalf("sweep events: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted events = %d, want 2", deleted)
	}
	remaining, _, err := eventStore.ListEvents(ctx, santaevents.ExecutionEventListParams{
		EventListParams: santaevents.EventListParams{HostID: host.ID},
	})
	if err != nil {
		t.Fatalf("list remaining events: %v", err)
	}
	if len(remaining) != 1 || remaining[0].Decision != santaevents.ExecutionDecisionBlockCertificate {
		t.Fatalf("remaining events = %+v, want newest event", remaining)
	}
}

func TestEventUploadDeduplicatesSigningChainsAcrossConcurrentUploads(t *testing.T) {
	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewSyncService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
		Heartbeats:     heartbeats.NewStore(db),
	})

	if _, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-concurrent-chain-host"},
		OrbitNodeKey: "santa-concurrent-chain-orbit",
	}); err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, sha := range []string{"concurrent-a", "concurrent-b"} {
		wg.Add(1)
		go func(sha string) {
			defer wg.Done()
			_, err := service.EventUpload(ctx, "santa-concurrent-chain-host", heartbeats.Contact{}, santa.EventUploadRequest{
				Events: []santaevents.ExecutionEventInput{{
					FileSHA256:   sha,
					FileName:     sha,
					OccurredAt:   time.Date(2026, 5, 23, 14, 0, 0, 0, time.UTC),
					Decision:     santaevents.ExecutionDecisionAllowBinary,
					SigningChain: santaTestSigningChain(),
				}},
			})
			errs <- err
		}(sha)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("event upload: %v", err)
		}
	}

	var chainCount int
	if err := db.Pool().QueryRow(ctx, `SELECT count(*) FROM santa_signing_chains`).Scan(&chainCount); err != nil {
		t.Fatalf("count signing chains: %v", err)
	}
	if chainCount != 1 {
		t.Fatalf("signing chain count = %d, want 1", chainCount)
	}
	var linkCount int
	if err := db.Pool().
		QueryRow(ctx, `SELECT count(*) FROM santa_executable_signing_chains`).
		Scan(&linkCount); err != nil {
		t.Fatalf("count signing chain links: %v", err)
	}
	if linkCount != 2 {
		t.Fatalf("signing chain link count = %d, want 2", linkCount)
	}
}

func TestEventUploadAvoidsDeadlocksAcrossReversedBatches(t *testing.T) {
	t.Run("executables", func(t *testing.T) {
		first := []santaevents.ExecutionEventInput{
			concurrentExecutionEvent("executable-a", ""),
			concurrentExecutionEvent("executable-b", ""),
		}
		second := []santaevents.ExecutionEventInput{
			concurrentExecutionEvent("executable-b", ""),
			concurrentExecutionEvent("executable-a", ""),
		}
		assertConcurrentEventUploads(t, "santa_executables", first, second)
	})

	t.Run("bundles", func(t *testing.T) {
		first := []santaevents.ExecutionEventInput{
			concurrentExecutionEvent("first-a", "bundle-a"),
			concurrentExecutionEvent("first-b", "bundle-b"),
		}
		second := []santaevents.ExecutionEventInput{
			concurrentExecutionEvent("second-b", "bundle-b"),
			concurrentExecutionEvent("second-a", "bundle-a"),
		}
		assertConcurrentEventUploads(t, "santa_bundles", first, second)
	})

	t.Run("signing chains", func(t *testing.T) {
		first := []santaevents.ExecutionEventInput{
			concurrentSignedExecutionEvent("first-a", "certificate-a"),
			concurrentSignedExecutionEvent("first-b", "certificate-b"),
		}
		second := []santaevents.ExecutionEventInput{
			concurrentSignedExecutionEvent("second-b", "certificate-b"),
			concurrentSignedExecutionEvent("second-a", "certificate-a"),
		}
		assertConcurrentEventUploads(t, "santa_signing_chains", first, second)
	})

	t.Run("certificates", func(t *testing.T) {
		first := concurrentExecutionEvent("first", "")
		first.SigningChain = []santaevents.CertificateInput{
			{SHA256: "certificate-a"},
			{SHA256: "certificate-b"},
		}
		second := concurrentExecutionEvent("second", "")
		second.SigningChain = []santaevents.CertificateInput{
			{SHA256: "certificate-b"},
			{SHA256: "certificate-a"},
		}
		assertConcurrentEventUploads(
			t,
			"santa_certificates",
			[]santaevents.ExecutionEventInput{first},
			[]santaevents.ExecutionEventInput{second},
		)
	})
}

func assertConcurrentEventUploads(
	t *testing.T,
	pauseTable string,
	first []santaevents.ExecutionEventInput,
	second []santaevents.ExecutionEventInput,
) {
	t.Helper()

	db, ctx := testdb.Open(t)
	hostStore := hosts.NewStore(db)
	eventStore := santaevents.NewStore(db)
	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "santa-reversed-batches-" + pauseTable},
		OrbitNodeKey: "santa-reversed-batches-orbit-" + pauseTable,
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	const gateKey int64 = 763218904
	triggerSQL := fmt.Sprintf(`
CREATE FUNCTION santa_test_pause_shared_write()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	PERFORM pg_advisory_xact_lock_shared(%d);
	RETURN NEW;
END
$$;

CREATE TRIGGER santa_test_pause_shared_write
AFTER INSERT OR UPDATE ON %s
FOR EACH ROW EXECUTE FUNCTION santa_test_pause_shared_write()`, gateKey, pauseTable)
	if _, err := db.Pool().Exec(ctx, triggerSQL); err != nil {
		t.Fatalf("create pause trigger: %v", err)
	}

	gate, err := db.Pool().Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire gate connection: %v", err)
	}
	gateLocked := false
	defer func() {
		// A failed test must not return a pooled session with its advisory lock held.
		if gateLocked {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			_, _ = gate.Exec(cleanupCtx, `SELECT pg_advisory_unlock($1)`, gateKey)
			cancel()
		}
		gate.Release()
	}()
	if _, err := gate.Exec(ctx, `SELECT pg_advisory_lock($1)`, gateKey); err != nil {
		t.Fatalf("lock upload gate: %v", err)
	}
	gateLocked = true

	// The trigger pauses each upload after its first shared-row write. Reversed
	// lock order then forces the old implementation into a circular wait,
	// while canonical order makes both uploads contend on the same first row.
	uploadCtx, cancelUploads := context.WithTimeout(ctx, 5*time.Second)
	defer cancelUploads()
	errs := make(chan error, 2)
	for _, upload := range [][]santaevents.ExecutionEventInput{first, second} {
		go func(events []santaevents.ExecutionEventInput) {
			_, err := eventStore.IngestEvents(uploadCtx, host.ID, events, nil, nil)
			errs <- err
		}(upload)
	}

	uploadsPaused, waitErr := waitForLockWaiters(uploadCtx, db, 2, 2*time.Second)
	if _, err := gate.Exec(ctx, `SELECT pg_advisory_unlock($1)`, gateKey); err != nil {
		cancelUploads()
		t.Fatalf("unlock upload gate: %v", err)
	}
	gateLocked = false

	if waitErr != nil {
		cancelUploads()
		t.Fatalf("wait for concurrent uploads: %v", waitErr)
	}
	if !uploadsPaused {
		cancelUploads()
		t.Fatal("concurrent uploads did not both reach the pause point")
	}
	for range 2 {
		select {
		case err := <-errs:
			if err != nil {
				t.Errorf("ingest reversed upload: %v", err)
			}
		case <-uploadCtx.Done():
			t.Fatalf("wait for reversed uploads: %v", uploadCtx.Err())
		}
	}

	assertHostExecutionEventCount(t, db, host.ID, len(first)+len(second))
}

func assertHostExecutionEventCount(t *testing.T, db *database.DB, hostID int64, want int) {
	t.Helper()

	var count int
	if err := db.Pool().
		QueryRow(t.Context(), `SELECT count(*) FROM santa_execution_events WHERE host_id = $1`, hostID).
		Scan(&count); err != nil {
		t.Fatalf("count execution events: %v", err)
	}
	if count != want {
		t.Fatalf("execution event count = %d, want %d", count, want)
	}
}

func waitForLockWaiters(
	ctx context.Context,
	db *database.DB,
	wanted int,
	limit time.Duration,
) (bool, error) {
	deadline := time.NewTimer(limit)
	defer deadline.Stop()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		var count int
		if err := db.Pool().QueryRow(ctx, `
SELECT count(*)
FROM pg_stat_activity
WHERE datname = current_database()
  AND wait_event_type = 'Lock'`).Scan(&count); err != nil {
			return false, err
		}
		if count >= wanted {
			return true, nil
		}

		select {
		case <-deadline.C:
			return false, nil
		case <-ticker.C:
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
}

func concurrentExecutionEvent(fileSHA256 string, bundleHash string) santaevents.ExecutionEventInput {
	return santaevents.ExecutionEventInput{
		FileSHA256:        fileSHA256,
		FileName:          fileSHA256,
		OccurredAt:        time.Date(2026, 5, 23, 14, 0, 0, 0, time.UTC),
		Decision:          santaevents.ExecutionDecisionAllowBinary,
		BundleHash:        bundleHash,
		BundleBinaryCount: 2,
	}
}

func concurrentSignedExecutionEvent(fileSHA256 string, certificateSHA256 string) santaevents.ExecutionEventInput {
	event := concurrentExecutionEvent(fileSHA256, "")
	event.SigningChain = []santaevents.CertificateInput{{SHA256: certificateSHA256}}
	return event
}

func santaTestSigningChain() []santaevents.CertificateInput {
	return []santaevents.CertificateInput{
		{SHA256: "leaf-sha", CommonName: "Leaf", Org: "Example", OU: "Engineering", ValidFrom: 1, ValidUntil: 2},
		{SHA256: "root-sha", CommonName: "Root", Org: "Example", OU: "Security", ValidFrom: 3, ValidUntil: 4},
	}
}
