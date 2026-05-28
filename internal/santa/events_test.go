package santa_test

import (
	"slices"
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
				FileSHA256:           "sha256-a",
				FilePath:             "/Applications/Example.app/Contents/MacOS/Example",
				FileName:             "Example",
				ExecutingUser:        "alice",
				ExecutionTimeSeconds: float64(occurredAt.Unix()),
				LoggedInUsers:        []string{" bob ", "alice", "bob", ""},
				CurrentSessions:      []string{"console", "ssh", "console"},
				Decision:             santaevents.ExecutionDecisionBlockBinary,
				BundleID:             "com.example.old",
				BundlePath:           "/Applications/Example.app",
				SigningID:            "TEAMID:com.example.old",
				TeamID:               "TEAMID",
				CDHash:               "old-cdhash",
				SigningChain:         santaTestSigningChain(),
			},
			{
				FileSHA256:    "sha256-a",
				FilePath:      "/Applications/Example.app/Contents/MacOS/Example",
				FileName:      "Example Renamed",
				ExecutingUser: "bob",
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

	items, _, err := eventStore.ListEvents(ctx, santaevents.EventListParams{HostID: host.ID})
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
	if blockEvent.OccurredAt == nil || !blockEvent.OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred_at = %v, want %v", blockEvent.OccurredAt, occurredAt)
	}
	if !slices.Equal(blockEvent.LoggedInUsers, []string{"bob", "alice"}) {
		t.Fatalf("logged_in_users = %v, want client order", blockEvent.LoggedInUsers)
	}
	if !slices.Equal(blockEvent.CurrentSessions, []string{"console", "ssh"}) {
		t.Fatalf("current_sessions = %v, want client order", blockEvent.CurrentSessions)
	}
	if allowEvent.OccurredAt != nil {
		t.Fatalf("zero execution time stored occurred_at = %v, want nil", allowEvent.OccurredAt)
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
	var linkCount int
	if err := db.Pool().
		QueryRow(ctx, `SELECT count(*) FROM santa_executable_signing_chains`).
		Scan(&linkCount); err != nil {
		t.Fatalf("count signing chain links: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("signing chain link count = %d, want 1", linkCount)
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
				FileSHA256:           string(rune('a' + i)),
				FileName:             string(rune('A' + i)),
				ExecutionTimeSeconds: float64(base.Add(time.Duration(i) * time.Minute).Unix()),
				Decision:             decision,
			}},
		})
		if err != nil {
			t.Fatalf("event upload %d: %v", i, err)
		}
	}

	firstPage, count, err := eventStore.ListEvents(
		ctx,
		santaevents.EventListParams{HostID: host.ID, ListParams: dbutil.ListParams{PageSize: 2}},
	)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(firstPage) != 2 || count != 3 {
		t.Fatalf("first page = %+v count=%d, want two items and count 3", firstPage, count)
	}
	secondPage, _, err := eventStore.ListEvents(
		ctx,
		santaevents.EventListParams{
			HostID:     host.ID,
			ListParams: dbutil.ListParams{PageSize: 2, PageIndex: 1},
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
		santaevents.EventListParams{
			HostID:    host.ID,
			Decisions: []santaevents.DecisionFilter{santaevents.DecisionFilterBlocked},
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
		santaevents.EventListParams{
			ListParams: dbutil.ListParams{Q: "B"},
			Decisions:  []santaevents.DecisionFilter{santaevents.DecisionFilterAllowed, "block_certificate"},
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
	remaining, _, err := eventStore.ListEvents(ctx, santaevents.EventListParams{HostID: host.ID})
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
