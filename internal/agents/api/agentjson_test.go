package agentapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteEncodesJSONResponse(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusAccepted, struct {
		OK bool `json:"ok"`
	}{OK: true})

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := rr.Body.String(); got != "{\"ok\":true}\n" {
		t.Fatalf("body = %q, want JSON object with trailing newline", got)
	}
}

func TestWriteErrorUsesAgentErrorShape(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	writeError(rr, http.StatusUnauthorized, "invalid node key")

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if got := rr.Body.String(); got != "{\"error\":\"invalid node key\"}\n" {
		t.Fatalf("body = %q, want agent error JSON", got)
	}
}
