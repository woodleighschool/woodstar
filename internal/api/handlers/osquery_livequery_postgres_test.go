//go:build postgres

package handlers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

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
	api := humachi.New(router, testHumaConfig())
	registerLiveQueries(api, api, storeB, nil, discardLogger())
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

func TestDeleteLiveQueryStopsSharedRun(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := livequery.NewStore(db)
	handle, err := store.Start(ctx, "select 1", []livequery.Target{{HostID: 42, HostName: "mac-42"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	router := chi.NewRouter()
	api := humachi.New(router, testHumaConfig())
	registerLiveQueries(api, api, store, nil, discardLogger())

	recorder := httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("/api/osquery/live-queries/%d", handle.ID), nil),
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

	recorder = httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(ctx, http.MethodDelete, "/api/osquery/live-queries/999999", nil),
	)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d; body = %q", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
}

func TestLiveQueryStreamReturnsNotFoundBeforeStreaming(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := livequery.NewStore(db)
	router := chi.NewRouter()
	api := humachi.New(router, testHumaConfig())
	registerLiveQueries(api, api, store, nil, discardLogger())

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
