package apitypes

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestPageMarshalsEmptyItemsAsArray(t *testing.T) {
	payload, err := json.Marshal(Page[int]{Items: nil, Count: 0})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(payload); !strings.Contains(got, `"items":[]`) {
		t.Fatalf("got %s, want items: []", got)
	}
}

func TestPageMarshalsItems(t *testing.T) {
	payload, err := json.Marshal(Page[int]{Items: []int{1, 2}, Count: 2})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(payload)
	if !strings.Contains(got, `"items":[1,2]`) || !strings.Contains(got, `"count":2`) {
		t.Fatalf("got %s", got)
	}
}

func TestResourceMutationErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not found", err: dbutil.ErrNotFound, wantStatus: 404},
		{name: "already exists", err: dbutil.ErrAlreadyExists, wantStatus: 409},
		{name: "validation", err: dbutil.ErrInvalidInput, wantStatus: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mapped := ResourceMutationError("resource", tt.err)
			var status huma.StatusError
			if !errors.As(mapped, &status) {
				t.Fatalf("not a huma.StatusError: %v", mapped)
			}
			if status.GetStatus() != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status.GetStatus(), tt.wantStatus)
			}
		})
	}
}
