package munki

import (
	"context"
	"strings"
	"testing"
)

func TestDetailIngestorIngestEnvelope(t *testing.T) {
	tests := []struct {
		name      string
		envelope  EnvelopeInput
		wantCall  bool
		complete  bool
		hasReport bool
		wantError string
		wantItems int
	}{
		{
			name: "complete report",
			envelope: EnvelopeInput{
				Info: QueryResult{Present: true, Rows: []map[string]string{{
					"version": "7.1", "errors": "first;second", "start_time": "2026-08-02 10:00:00 +1000",
				}}},
				Installs: QueryResult{Present: true, Rows: []map[string]string{{"name": "Firefox", "installed": "true"}}},
			},
			wantCall: true, complete: true, hasReport: true, wantItems: 1,
		},
		{
			name: "complete no report",
			envelope: EnvelopeInput{
				Info:     QueryResult{Present: true},
				Installs: QueryResult{Present: true},
			},
			wantCall: true, complete: true,
		},
		{
			name: "info failed",
			envelope: EnvelopeInput{
				Info:     QueryResult{Present: true, Status: 1, Message: "extension unavailable"},
				Installs: QueryResult{Present: true},
			},
			wantCall: true, wantError: "munki_info: extension unavailable",
		},
		{
			name: "installs missing",
			envelope: EnvelopeInput{
				Info: QueryResult{Present: true},
			},
			wantCall: true, wantError: "munki_installs: missing result",
		},
		{
			name:     "family absent",
			envelope: EnvelopeInput{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &recordingEnvelopeStore{}
			ingestor := NewDetailIngestor(store)
			if err := ingestor.IngestEnvelope(context.Background(), 42, tt.envelope); err != nil {
				t.Fatalf("IngestEnvelope: %v", err)
			}
			assertIngestedEnvelope(t, store, tt.wantCall, tt.complete, tt.hasReport, tt.wantError, tt.wantItems)
		})
	}
}

func assertIngestedEnvelope(
	t *testing.T,
	store *recordingEnvelopeStore,
	wantCall bool,
	wantComplete bool,
	wantReport bool,
	wantError string,
	wantItems int,
) {
	t.Helper()
	if store.called != wantCall {
		t.Fatalf("ApplyEnvelope called = %t, want %t", store.called, wantCall)
	}
	if !wantCall {
		return
	}
	if store.result.HostID != 42 || store.result.AttemptedAt.IsZero() {
		t.Fatalf("result identity/attempt = %+v, want host and non-zero attempt", store.result)
	}
	if store.result.Complete != wantComplete || store.result.HasReport != wantReport {
		t.Fatalf("result = %+v, want complete=%t has_report=%t", store.result, wantComplete, wantReport)
	}
	if store.result.CollectionError != wantError {
		t.Fatalf("collection error = %q, want %q", store.result.CollectionError, wantError)
	}
	if len(store.result.Items) != wantItems {
		t.Fatalf("items = %#v, want %d", store.result.Items, wantItems)
	}
	if wantReport && (strings.Join(store.result.Observation.Errors, ";") != "first;second" || store.result.Observation.RunStartedAt == nil) {
		t.Fatalf("observation = %+v, want parsed diagnostics and timestamp", store.result.Observation)
	}
}

type recordingEnvelopeStore struct {
	called bool
	result EnvelopeResult
}

func (s *recordingEnvelopeStore) ApplyEnvelope(_ context.Context, result EnvelopeResult) error {
	s.called = true
	s.result = result
	return nil
}
