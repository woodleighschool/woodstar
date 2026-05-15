package handlers

import (
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
)

func TestLabelListInputParams(t *testing.T) {
	input := labelListInput{
		Q:              " mac ",
		Page:           2,
		PerPage:        25,
		OrderKey:       "name",
		OrderDirection: "desc",
		LabelType:      "regular",
		MembershipType: "dynamic",
		Platform:       " darwin ",
	}

	got := input.params()
	if got.Q != "mac" || got.Page != 2 || got.PerPage != 25 {
		t.Fatalf("list params = %#v", got.ListParams)
	}
	if got.LabelType != labels.LabelTypeRegular {
		t.Fatalf("LabelType = %q, want regular", got.LabelType)
	}
	if got.LabelMembershipType != labels.LabelMembershipTypeDynamic {
		t.Fatalf("LabelMembershipType = %q, want dynamic", got.LabelMembershipType)
	}
	if got.Platform != "darwin" {
		t.Fatalf("platform.Platform = %q, want darwin", got.Platform)
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
			mapped := resourceMutationError("label", tt.err)
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
