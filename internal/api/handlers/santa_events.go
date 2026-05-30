package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
)

const (
	santaEventResource         = "Santa event"
	santaEventIDPath           = "/api/santa/events/{id}"
	santaFileAccessEventPath   = "/api/santa/file-access-events"
	santaFileAccessEventIDPath = "/api/santa/file-access-events/{id}"
)

type santaEventListInput struct {
	ListQueryInput
	HostID    int64                        `query:"host_id,omitempty"`
	Decisions []santaevents.DecisionFilter `query:"decisions,omitempty"`
	Since     time.Time                    `query:"since,omitempty"`
	User      string                       `query:"user,omitempty"`
}

type santaEventListOutput struct {
	Body Page[santaevents.ExecutionEvent]
}

type santaEventGetInput struct {
	ID int64 `path:"id"`
}

type santaEventGetOutput struct {
	Body *santaevents.ExecutionEvent
}

type santaFileAccessEventListInput struct {
	ListQueryInput
	HostID    int64                            `query:"host_id,omitempty"`
	Decisions []santaevents.FileAccessDecision `query:"decisions,omitempty"`
	Since     time.Time                        `query:"since,omitempty"`
}

type santaFileAccessEventListOutput struct {
	Body Page[santaevents.FileAccessEvent]
}

type santaFileAccessEventGetInput struct {
	ID int64 `path:"id"`
}

type santaFileAccessEventGetOutput struct {
	Body *santaevents.FileAccessEvent
}

func (input santaEventListInput) params() santaevents.ExecutionEventListParams {
	var since *time.Time
	if !input.Since.IsZero() {
		since = &input.Since
	}
	return santaevents.ExecutionEventListParams{
		EventListParams: santaevents.EventListParams{
			ListParams: input.ListQueryInput.params(),
			HostID:     input.HostID,
			Since:      since,
		},
		Decisions: input.Decisions,
		User:      input.User,
	}
}

func (input santaFileAccessEventListInput) params() santaevents.FileAccessEventListParams {
	var since *time.Time
	if !input.Since.IsZero() {
		since = &input.Since
	}
	return santaevents.FileAccessEventListParams{
		EventListParams: santaevents.EventListParams{
			ListParams: input.ListQueryInput.params(),
			HostID:     input.HostID,
			Since:      since,
		},
		Decisions: input.Decisions,
	}
}

func RegisterSantaEvents(api huma.API, store *santaevents.Store) {
	registerListSantaEvents(api, store)
	registerGetSantaEvent(api, store)
	registerListSantaFileAccessEvents(api, store)
	registerGetSantaFileAccessEvent(api, store)
}

func registerListSantaEvents(api huma.API, store *santaevents.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-events",
		Method:      http.MethodGet,
		Path:        "/api/santa/events",
		Tags:        []string{santaTag},
		Summary:     "List Santa execution events",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaEventListInput) (*santaEventListOutput, error) {
		events, count, err := store.ListEvents(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError("Santa event", err)
		}
		return &santaEventListOutput{Body: Page[santaevents.ExecutionEvent]{Items: events, Count: count}}, nil
	})
}

func registerGetSantaEvent(api huma.API, store *santaevents.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-event",
		Method:      http.MethodGet,
		Path:        santaEventIDPath,
		Tags:        []string{santaTag},
		Summary:     "Get a Santa execution event",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaEventGetInput) (*santaEventGetOutput, error) {
		event, err := store.GetExecutionEvent(ctx, input.ID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("Santa event not found")
		}
		if err != nil {
			return nil, resourceMutationError(santaEventResource, err)
		}
		return &santaEventGetOutput{Body: event}, nil
	})
}

func registerListSantaFileAccessEvents(api huma.API, store *santaevents.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-file-access-events",
		Method:      http.MethodGet,
		Path:        santaFileAccessEventPath,
		Tags:        []string{santaTag},
		Summary:     "List Santa file access events",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaFileAccessEventListInput) (*santaFileAccessEventListOutput, error) {
		events, count, err := store.ListFileAccessEvents(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError("Santa file access event", err)
		}
		return &santaFileAccessEventListOutput{
			Body: Page[santaevents.FileAccessEvent]{Items: events, Count: count},
		}, nil
	})
}

func registerGetSantaFileAccessEvent(api huma.API, store *santaevents.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-file-access-event",
		Method:      http.MethodGet,
		Path:        santaFileAccessEventIDPath,
		Tags:        []string{santaTag},
		Summary:     "Get a Santa file access event",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaFileAccessEventGetInput) (*santaFileAccessEventGetOutput, error) {
		event, err := store.GetFileAccessEvent(ctx, input.ID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("Santa file access event not found")
		}
		if err != nil {
			return nil, resourceMutationError("Santa file access event", err)
		}
		return &santaFileAccessEventGetOutput{Body: event}, nil
	})
}
