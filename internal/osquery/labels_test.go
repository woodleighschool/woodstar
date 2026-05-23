package osquery

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestHandleLabelResultStatusFilter(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	rows := []map[string]string{{"col": "val"}}

	t.Run("failed status skips accumulation", func(t *testing.T) {
		pass := &labelDispatchPass{}
		handleLabelResult(context.Background(), logger, 1, "10", rows, json.RawMessage("1"), "", pass)
		if len(pass.results) != 0 {
			t.Fatalf("labelResults = %v, want empty", pass.results)
		}
	})

	t.Run("success status appends matched result", func(t *testing.T) {
		pass := &labelDispatchPass{}
		handleLabelResult(context.Background(), logger, 1, "10", rows, json.RawMessage("0"), "", pass)
		if len(pass.results) != 1 {
			t.Fatalf("labelResults len = %d, want 1", len(pass.results))
		}
		if pass.results[0].LabelID != 10 || !pass.results[0].Matched {
			t.Fatalf("labelResults[0] = %+v, want {LabelID:10 Matched:true}", pass.results[0])
		}
	})
}
