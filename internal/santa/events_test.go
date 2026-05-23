package santa_test

import (
	"slices"
	"testing"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa"
)

func TestEventUploadIngestsExecutionEventsAndUpdatesExecutableMetadata(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	store := santa.NewStore(db)
	service := santa.NewService(store)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID:   "santa-events-host",
		HardwareSerial: "SANTAEVENTS",
		OrbitNodeKey:   "santa-events-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	occurredAt := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)
	_, err = service.HandleEventUpload(ctx, "santa-events-host", &syncv1.EventUploadRequest{
		MachineId: "santa-events-host",
		Events: []*syncv1.Event{
			{
				FileSha256:      "sha256-a",
				FilePath:        "/Applications/Example.app/Contents/MacOS/Example",
				FileName:        "Example",
				ExecutingUser:   "alice",
				ExecutionTime:   float64(occurredAt.Unix()),
				LoggedInUsers:   []string{" bob ", "alice", "bob", ""},
				CurrentSessions: []string{"console", "ssh", "console"},
				Decision:        syncv1.Decision_BLOCK_BINARY,
				FileBundleId:    "com.example.old",
				FileBundlePath:  "/Applications/Example.app",
				SigningId:       "TEAMID:com.example.old",
				TeamId:          "TEAMID",
				Cdhash:          "old-cdhash",
				SigningChain:    santaTestSigningChain(),
			},
			{
				FileSha256:     "sha256-a",
				FilePath:       "/Applications/Example.app/Contents/MacOS/Example",
				FileName:       "Example Renamed",
				ExecutingUser:  "bob",
				Decision:       syncv1.Decision_ALLOW_BINARY,
				FileBundleId:   "com.example.new",
				FileBundlePath: "/Applications/Example.app",
				SigningId:      "TEAMID:com.example.new",
				TeamId:         "TEAMID",
				Cdhash:         "new-cdhash",
			},
		},
		FileAccessEvents: []*syncv1.FileAccessEvent{{Target: "/Users/alice/Documents/private.txt"}},
	})
	if err != nil {
		t.Fatalf("event upload: %v", err)
	}

	page, err := store.ListEvents(ctx, santa.EventListParams{HostID: host.ID, Limit: 10})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("events = %+v, want two execution events", page.Items)
	}
	allowEvent := santa.ExecutionEvent{}
	blockEvent := santa.ExecutionEvent{}
	for _, event := range page.Items {
		if event.Decision == santa.ExecutionDecisionAllowBinary {
			allowEvent = event
		}
		if event.Decision == santa.ExecutionDecisionBlockBinary {
			blockEvent = event
		}
	}
	if allowEvent.ID == 0 || blockEvent.ID == 0 {
		t.Fatalf("events = %+v, want allow_binary and block_binary", page.Items)
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
	service := santa.NewService(store)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-event-list-host",
		OrbitNodeKey: "santa-event-list-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	base := time.Date(2026, 5, 23, 13, 0, 0, 0, time.UTC)
	for i, decision := range []syncv1.Decision{
		syncv1.Decision_BLOCK_BINARY,
		syncv1.Decision_ALLOW_BINARY,
		syncv1.Decision_BLOCK_CERTIFICATE,
	} {
		_, err := service.HandleEventUpload(ctx, "santa-event-list-host", &syncv1.EventUploadRequest{
			MachineId: "santa-event-list-host",
			Events: []*syncv1.Event{{
				FileSha256:    string(rune('a' + i)),
				FileName:      string(rune('A' + i)),
				ExecutionTime: float64(base.Add(time.Duration(i) * time.Minute).Unix()),
				Decision:      decision,
			}},
		})
		if err != nil {
			t.Fatalf("event upload %d: %v", i, err)
		}
	}

	firstPage, err := store.ListEvents(ctx, santa.EventListParams{HostID: host.ID, Limit: 2})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(firstPage.Items) != 2 || firstPage.NextCursor == "" {
		t.Fatalf("first page = %+v, want two items and cursor", firstPage)
	}
	secondPage, err := store.ListEvents(
		ctx,
		santa.EventListParams{HostID: host.ID, Limit: 2, After: firstPage.NextCursor},
	)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(secondPage.Items) != 1 || secondPage.Items[0].Decision != santa.ExecutionDecisionBlockBinary {
		t.Fatalf("second page = %+v, want oldest blocked binary event", secondPage.Items)
	}

	blocked, err := store.ListEvents(
		ctx,
		santa.EventListParams{HostID: host.ID, Decision: santa.EventDecisionClassBlocked, Limit: 10},
	)
	if err != nil {
		t.Fatalf("list blocked events: %v", err)
	}
	if len(blocked.Items) != 2 {
		t.Fatalf("blocked events = %+v, want two", blocked.Items)
	}

	deleted, err := store.SweepEventsBefore(ctx, base.Add(90*time.Second))
	if err != nil {
		t.Fatalf("sweep events: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted events = %d, want 2", deleted)
	}
	remaining, err := store.ListEvents(ctx, santa.EventListParams{HostID: host.ID, Limit: 10})
	if err != nil {
		t.Fatalf("list remaining events: %v", err)
	}
	if len(remaining.Items) != 1 || remaining.Items[0].Decision != santa.ExecutionDecisionBlockCertificate {
		t.Fatalf("remaining events = %+v, want newest event", remaining.Items)
	}
}

func santaTestSigningChain() []*syncv1.Certificate {
	return []*syncv1.Certificate{
		{Sha256: "leaf-sha", Cn: "Leaf", Org: "Example", Ou: "Engineering", ValidFrom: 1, ValidUntil: 2},
		{Sha256: "root-sha", Cn: "Root", Org: "Example", Ou: "Security", ValidFrom: 3, ValidUntil: 4},
	}
}
