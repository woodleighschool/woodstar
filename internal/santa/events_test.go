package santa_test

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

func TestEventUploadIngestsExecutionEventsAndUpdatesExecutableMetadata(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	store := santa.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewService(santa.Dependencies{
		HostStore:      store,
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
	})

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID:   "santa-events-host",
		HardwareSerial: "SANTAEVENTS",
		OrbitNodeKey:   "santa-events-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	occurredAt := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)
	_, err = service.EventUpload(ctx, "santa-events-host", santa.EventUploadRequest{
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
				FileSHA256:    "sha256-a",
				FilePath:      "/Applications/Example.app/Contents/MacOS/Example",
				FileName:      "Example Renamed",
				ExecutingUser: "bob",
				OccurredAt:    occurredAt.Add(time.Second),
				Decision:      santaevents.ExecutionDecisionAllowBinary,
				BundleID:      "com.example.new",
				BundlePath:    "/Applications/Example.app",
				SigningID:     "TEAMID:com.example.new",
				TeamID:        "TEAMID",
				CDHash:        "new-cdhash",
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
	allowEvent := santaevents.ExecutionEvent{}
	blockEvent := santaevents.ExecutionEvent{}
	for _, event := range items {
		if event.Decision == santaevents.ExecutionDecisionAllowBinary {
			allowEvent = event
		}
		if event.Decision == santaevents.ExecutionDecisionBlockBinary {
			blockEvent = event
		}
	}
	if allowEvent.ID == 0 || blockEvent.ID == 0 {
		t.Fatalf("events = %+v, want allow_binary and block_binary", items)
	}
	if allowEvent.Executable.FileName != "Example Renamed" ||
		allowEvent.Executable.BundleID != "com.example.new" ||
		allowEvent.Executable.SigningID != "TEAMID:com.example.new" ||
		allowEvent.Executable.CDHash != "new-cdhash" {
		t.Fatalf("executable metadata was not updated: %+v", allowEvent.Executable)
	}
	if !blockEvent.OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred_at = %v, want %v", blockEvent.OccurredAt, occurredAt)
	}
	if !slices.Equal(blockEvent.LoggedInUsers, []string{"bob", "alice"}) {
		t.Fatalf("logged_in_users = %v, want client order", blockEvent.LoggedInUsers)
	}
	if !slices.Equal(blockEvent.CurrentSessions, []string{"console", "ssh"}) {
		t.Fatalf("current_sessions = %v, want client order", blockEvent.CurrentSessions)
	}
	if !allowEvent.OccurredAt.Equal(occurredAt.Add(time.Second)) {
		t.Fatalf("occurred_at = %v, want %v", allowEvent.OccurredAt, occurredAt.Add(time.Second))
	}
	if len(allowEvent.LoggedInUsers) != 0 {
		t.Fatalf("omitted logged_in_users = %v, want empty array", allowEvent.LoggedInUsers)
	}
	if len(allowEvent.CurrentSessions) != 0 {
		t.Fatalf("omitted current_sessions = %v, want empty array", allowEvent.CurrentSessions)
	}

	var chainCount int
	if err := db.Pool().QueryRow(ctx, `SELECT count(*) FROM santa_signing_chains`).Scan(&chainCount); err != nil {
		t.Fatalf("count signing chains: %v", err)
	}
	if chainCount != 1 {
		t.Fatalf("signing chain count = %d, want 1", chainCount)
	}
	var certificateCount int
	if err := db.Pool().QueryRow(ctx, `SELECT count(*) FROM santa_certificates`).Scan(&certificateCount); err != nil {
		t.Fatalf("count certificates: %v", err)
	}
	if certificateCount != 2 {
		t.Fatalf("certificate count = %d, want 2", certificateCount)
	}
	var chainEntryCount int
	if err := db.Pool().
		QueryRow(ctx, `SELECT count(*) FROM santa_signing_chain_entries`).
		Scan(&chainEntryCount); err != nil {
		t.Fatalf("count signing chain entries: %v", err)
	}
	if chainEntryCount != 2 {
		t.Fatalf("signing chain entry count = %d, want 2", chainEntryCount)
	}
	var linkCount int
	if err := db.Pool().
		QueryRow(ctx, `SELECT count(*) FROM santa_executable_signing_chains`).
		Scan(&linkCount); err != nil {
		t.Fatalf("count signing chain links: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("signing chain link count = %d, want 1", linkCount)
	}

	detail, err := eventStore.GetExecutionEvent(ctx, blockEvent.ID)
	if err != nil {
		t.Fatalf("get execution event: %v", err)
	}
	if detail.Host.ID != host.ID || detail.Host.DisplayName != host.DisplayName {
		t.Fatalf("detail host = %+v, want host %d/%q", detail.Host, host.ID, host.DisplayName)
	}
	if len(detail.Executable.SigningChain) != 2 ||
		detail.Executable.SigningChain[0].CommonName != "Leaf" ||
		detail.Executable.SigningChain[1].SHA256 != "root-sha" {
		t.Fatalf("detail signing chain = %+v, want full chain", detail.Executable.SigningChain)
	}
}

func TestEventUploadRequestsAndCollectsBundleBinaries(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
	})

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-bundle-events-host",
		OrbitNodeKey: "santa-bundle-events-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	bundleHash := strings.Repeat("b", 64)
	firstResponse, err := service.EventUpload(ctx, "santa-bundle-events-host", santa.EventUploadRequest{
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

	secondResponse, err := service.EventUpload(ctx, "santa-bundle-events-host", santa.EventUploadRequest{
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

func TestEventUploadRejectsEventsWithoutOccurrenceTime(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
	})

	if _, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-event-time-required-host",
		OrbitNodeKey: "santa-event-time-required-orbit",
	}); err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	if _, err := service.EventUpload(ctx, "santa-event-time-required-host", santa.EventUploadRequest{
		Events: []santaevents.ExecutionEventInput{{
			FileSHA256: "sha-without-time",
			Decision:   santaevents.ExecutionDecisionBlockBinary,
		}},
	}); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("event upload err = %v, want invalid input", err)
	}
}

func TestEventUploadIngestsFileAccessEvents(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
	})

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-file-access-host",
		Hostname:     "file-access.example.test",
		OrbitNodeKey: "santa-file-access-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	occurredAt := time.Date(2026, 5, 24, 9, 15, 0, 0, time.UTC)
	_, err = service.EventUpload(ctx, "santa-file-access-host", santa.EventUploadRequest{
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
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	store := santa.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewService(santa.Dependencies{
		HostStore:      store,
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
	})

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-event-list-host",
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
		_, err := service.EventUpload(ctx, "santa-event-list-host", santa.EventUploadRequest{
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
				ListParams: dbutil.ListParams{PageSize: 2},
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
				ListParams: dbutil.ListParams{PageSize: 2, PageIndex: 1},
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
				ListParams: dbutil.ListParams{Q: "B"},
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
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	eventStore := santaevents.NewStore(db)
	service := santa.NewService(santa.Dependencies{
		HostStore:      santa.NewStore(db),
		Configurations: configurations.NewStore(db),
		Events:         eventStore,
		Rules:          santarules.NewStore(db),
		Sync:           syncstate.NewStore(db),
	})

	if _, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-concurrent-chain-host",
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
			_, err := service.EventUpload(ctx, "santa-concurrent-chain-host", santa.EventUploadRequest{
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

func santaTestSigningChain() []santaevents.CertificateInput {
	return []santaevents.CertificateInput{
		{SHA256: "leaf-sha", CommonName: "Leaf", Org: "Example", OU: "Engineering", ValidFrom: 1, ValidUntil: 2},
		{SHA256: "root-sha", CommonName: "Root", Org: "Example", OU: "Security", ValidFrom: 3, ValidUntil: 4},
	}
}
