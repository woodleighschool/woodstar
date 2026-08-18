//go:build postgres

package httpapi

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestLiveQueryStreamFollowsResultsFromAnotherStore(t *testing.T) {
	db, ctx := testdb.Open(t)
	storeA := livequery.NewStore(db)
	storeB := livequery.NewStore(db)
	handle, err := storeA.Start(ctx, "select 1", []livequery.Target{{HostID: 4, HostName: "mac-4"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	router := chi.NewRouter()
	humaAPI := humachi.New(router, testHumaConfig())
	registerLiveQueries(humaAPI, humaAPI, storeB, nil, nil, discardLogger())
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	streamCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(
		streamCtx,
		http.MethodGet,
		fmt.Sprintf("%s/api/osquery/live-queries/%d/stream", server.URL, handle.ID),
		nil,
	)
	if err != nil {
		t.Fatalf("new stream request: %v", err)
	}
	request.Header.Set("Accept", "text/event-stream")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("stream status = %d, want %d; body = %q", response.StatusCode, http.StatusOK, body)
	}

	reader := bufio.NewReader(response.Body)
	pending := readSSEEvent(t, reader)
	if !strings.Contains(pending, `"host_id":4`) || !strings.Contains(pending, `"status":"pending"`) {
		t.Fatalf("pending event = %q", pending)
	}
	if err := storeA.RecordResult(ctx, livequery.Result{
		QueryID:  handle.ID,
		HostID:   4,
		HostName: "mac-4-renamed",
		Status:   livequery.StatusCollected,
		Rows:     []map[string]string{{"answer": "1"}},
	}); err != nil {
		t.Fatalf("RecordResult through store A: %v", err)
	}
	collected := readSSEEvent(t, reader)
	if !strings.Contains(collected, `"status":"collected"`) ||
		!strings.Contains(collected, `"host_name":"mac-4-renamed"`) ||
		!strings.Contains(collected, `"rows":[{"answer":"1"}]`) {
		t.Fatalf("collected event = %q", collected)
	}
	completed := readSSEEvent(t, reader)
	if !strings.Contains(completed, `"type":"completed"`) {
		t.Fatalf("completed event = %q", completed)
	}
}

func TestLiveQueryStreamReplaysCompletedResultsFromAnotherStore(t *testing.T) {
	db, ctx := testdb.Open(t)
	storeA := livequery.NewStore(db)
	storeB := livequery.NewStore(db)
	handle, err := storeA.Start(ctx, "select 1", []livequery.Target{{HostID: 4, HostName: "mac-4"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := storeA.RecordResult(ctx, livequery.Result{
		QueryID:  handle.ID,
		HostID:   4,
		HostName: "mac-4",
		Status:   livequery.StatusCollected,
		Rows:     []map[string]string{{"answer": "1"}},
	}); err != nil {
		t.Fatalf("RecordResult: %v", err)
	}

	router := chi.NewRouter()
	humaAPI := humachi.New(router, testHumaConfig())
	registerLiveQueries(humaAPI, humaAPI, storeB, nil, nil, discardLogger())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/api/osquery/live-queries/%d/stream", handle.ID),
		nil,
	)
	request.Header.Set("Accept", "text/event-stream")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	body := recorder.Body.String()
	snapshotAt := strings.Index(body, "event: snapshot")
	completedAt := strings.Index(body, "event: completed")
	if snapshotAt < 0 || completedAt < 0 || snapshotAt > completedAt {
		t.Fatalf("SSE body = %q, want final snapshot before completion", body)
	}
	for _, want := range []string{`"status":"collected"`, `"host_name":"mac-4"`, `"rows":[{"answer":"1"}]`} {
		if !strings.Contains(body, want) {
			t.Fatalf("SSE body = %q, want %q", body, want)
		}
	}
}

func TestReadLiveQuerySnapshotsTimesOutOnLockedRun(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := livequery.NewStore(db)
	handle, err := store.Start(ctx, "select 1", []livequery.Target{{HostID: 4, HostName: "mac-4"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin lock transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var lockedID int64
	if err := tx.QueryRow(ctx, `
SELECT id
FROM osquery_live_query_runs
WHERE id = $1
FOR UPDATE`, handle.ID).Scan(&lockedID); err != nil {
		t.Fatalf("lock run: %v", err)
	}

	started := time.Now()
	_, _, err = readLiveQuerySnapshots(ctx, store, handle.ID)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Snapshots error = %v, want context deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 2*liveQuerySnapshotTimeout {
		t.Fatalf("Snapshots took %s, want at most %s", elapsed, 2*liveQuerySnapshotTimeout)
	}
}

func TestDeleteLiveQueryStopsSharedRun(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := livequery.NewStore(db)
	handle, err := store.Start(ctx, "select 1", []livequery.Target{{HostID: 42, HostName: "mac-42"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	router := chi.NewRouter()
	humaAPI := humachi.New(router, testHumaConfig())
	activities := &recordingActivity{}
	registerLiveQueries(humaAPI, humaAPI, store, nil, activities, discardLogger())

	recorder := httptest.NewRecorder()
	requestCtx := ctxkeys.WithUser(ctx, &directory.User{ID: 9, Name: "Query Admin"})
	router.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(
			requestCtx, http.MethodDelete, fmt.Sprintf("/api/osquery/live-queries/%d", handle.ID), nil,
		),
	)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusNoContent, recorder.Body.String())
	}
	work, err := store.PendingForHost(ctx, 42)
	if err != nil {
		t.Fatalf("PendingForHost: %v", err)
	}
	if len(work) != 0 {
		t.Fatalf("pending work after DELETE = %+v, want none", work)
	}
	if len(activities.events) != 1 || activities.events[0].Action != activity.ActionLiveQueryStopped ||
		activities.events[0].Actor.UserID == nil || *activities.events[0].Actor.UserID != 9 ||
		activities.events[0].Subject.ID == nil || *activities.events[0].Subject.ID != handle.ID {
		t.Fatalf("activity = %+v, want one authenticated live-query stop event", activities.events)
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(ctx, http.MethodDelete, "/api/osquery/live-queries/999999", nil),
	)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d; body = %q", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if len(activities.events) != 1 {
		t.Fatalf("activity after missing query = %+v, want no additional event", activities.events)
	}
}

type recordingActivity struct {
	events []activity.NewEvent
}

func (recorder *recordingActivity) Record(_ context.Context, event activity.NewEvent) error {
	recorder.events = append(recorder.events, event)
	return nil
}

func TestLiveQueryStreamReturnsNotFoundBeforeStreaming(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := livequery.NewStore(db)
	router := chi.NewRouter()
	humaAPI := humachi.New(router, testHumaConfig())
	registerLiveQueries(humaAPI, humaAPI, store, nil, nil, discardLogger())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/osquery/live-queries/404/stream", nil)
	request.Header.Set("Accept", "text/event-stream")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want a regular error response", contentType)
	}
}

func readSSEEvent(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	var event strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE event: %v", err)
		}
		if line == "\n" {
			return event.String()
		}
		event.WriteString(line)
	}
}
