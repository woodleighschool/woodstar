package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
)

func TestLiveQueryStreamReturnsNotFoundBeforeStreaming(t *testing.T) {
	router := chi.NewRouter()
	api := humachi.New(router, testHumaConfig())
	registerLiveQueries(api, livequery.NewManager(), nil, discardLogger())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/live-queries/404/stream", nil)
	request.Header.Set("Accept", "text/event-stream")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want a regular error response", contentType)
	}
}

func TestLiveQueryStreamReplaysCompletedResults(t *testing.T) {
	manager := livequery.NewManager()
	handle := manager.Start("select 1", []int64{4})
	manager.RecordResult(livequery.Result{
		QueryID:  handle.ID,
		HostID:   4,
		HostName: "mac-4",
		Status:   livequery.StatusSuccess,
	})
	router := chi.NewRouter()
	api := humachi.New(router, testHumaConfig())
	registerLiveQueries(api, manager, nil, discardLogger())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/live-queries/1/stream", nil)
	request.Header.Set("Accept", "text/event-stream")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	body := recorder.Body.String()
	resultAt := strings.Index(body, "event: result")
	completedAt := strings.Index(body, "event: completed")
	if resultAt < 0 || completedAt < 0 || resultAt > completedAt {
		t.Fatalf("SSE body = %q, want replayed result before completion", body)
	}
}
