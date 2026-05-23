package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/santa"
)

const santaEventsTag = "Santa"

type santaEventListInput struct {
	HostID   string `query:"host_id,omitempty"`
	Decision string `query:"decision,omitempty"`
	Since    string `query:"since,omitempty"`
	Limit    int    `query:"limit,omitempty"`
	After    string `query:"after,omitempty"`
}

type santaEventListOutput struct {
	Body santa.EventPage
}

func (input santaEventListInput) params() (santa.EventListParams, error) {
	hostID, err := parseOptionalPositiveID(input.HostID, "host_id")
	if err != nil {
		return santa.EventListParams{}, err
	}
	var since *time.Time
	if input.Since != "" {
		parsed, err := time.Parse(time.RFC3339, input.Since)
		if err != nil {
			return santa.EventListParams{}, huma.Error400BadRequest(fmt.Sprintf("invalid since: %v", err))
		}
		since = &parsed
	}
	return santa.EventListParams{
		HostID:   hostID,
		Decision: santa.ExecutionDecision(input.Decision),
		Since:    since,
		Limit:    input.Limit,
		After:    input.After,
	}, nil
}

func RegisterSantaEvents(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-events",
		Method:      http.MethodGet,
		Path:        "/api/santa/events",
		Tags:        []string{santaEventsTag},
		Summary:     "List Santa execution events",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaEventListInput) (*santaEventListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		params, err := input.params()
		if err != nil {
			return nil, err
		}
		page, err := store.ListEvents(ctx, params)
		if err != nil {
			return nil, resourceMutationError("Santa event", err)
		}
		return &santaEventListOutput{Body: page}, nil
	})
}
