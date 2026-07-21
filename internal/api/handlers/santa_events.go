package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/events"
)

const (
	santaEventResource       = "Santa event"
	santaEventIDPath         = "/api/santa/events/{id}"
	santaFileAccessEventPath = "/api/santa/file-access-events"
	santaFileAccessIDPath    = "/api/santa/file-access-events/{id}"
)

type santaEventListInput struct {
	ListQueryInput

	HostID    int64                   `query:"host_id,omitempty"`
	Decisions []events.DecisionFilter `query:"decisions,omitempty"`
	Since     time.Time               `query:"since,omitempty"`
	User      string                  `query:"user,omitempty"`
}

type santaEventListOutput struct {
	Body Page[events.ExecutionEvent]
}

type santaEventGetInput struct {
	ID int64 `path:"id"`
}

type santaEventGetOutput struct {
	Body *events.ExecutionEvent
}

type santaFileAccessEventListInput struct {
	ListQueryInput

	HostID    int64                       `query:"host_id,omitempty"`
	Decisions []events.FileAccessDecision `query:"decisions,omitempty"`
	Since     time.Time                   `query:"since,omitempty"`
}

type santaFileAccessEventListOutput struct {
	Body Page[events.FileAccessEvent]
}

type santaFileAccessEventGetInput struct {
	ID int64 `path:"id"`
}

type santaFileAccessEventGetOutput struct {
	Body *events.FileAccessEvent
}

func (input santaEventListInput) params() events.ExecutionEventListParams {
	return events.ExecutionEventListParams{
		EventListParams: events.EventListParams{
			ListParams: input.ListQueryInput.params(),
			HostID:     input.HostID,
			Since:      input.Since,
		},
		Decisions: input.Decisions,
		User:      input.User,
	}
}

func (input santaFileAccessEventListInput) params() events.FileAccessEventListParams {
	return events.FileAccessEventListParams{
		EventListParams: events.EventListParams{
			ListParams: input.ListQueryInput.params(),
			HostID:     input.HostID,
			Since:      input.Since,
		},
		Decisions: input.Decisions,
	}
}

func registerSantaEvents(api huma.API, store *events.Store, logger *slog.Logger) {
	registerListSantaEvents(api, store, logger)
	registerGetSantaEvent(api, store, logger)
	registerListSantaFileAccessEvents(api, store, logger)
	registerGetSantaFileAccessEvent(api, store, logger)
}

func registerListSantaEvents(api huma.API, store *events.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-events",
		Method:      http.MethodGet,
		Path:        "/api/santa/events",
		Tags:        []string{santaEventsTag},
		Summary:     "List execution events",
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *santaEventListInput) (*santaEventListOutput, error) {
		rows, count, err := store.ListEvents(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-santa-events", "Santa event", err)
		}
		return &santaEventListOutput{Body: Page[events.ExecutionEvent]{Items: rows, Count: count}}, nil
	})
}

func registerGetSantaEvent(api huma.API, store *events.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-event",
		Method:      http.MethodGet,
		Path:        santaEventIDPath,
		Tags:        []string{santaEventsTag},
		Summary:     "Get an execution event",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *santaEventGetInput) (*santaEventGetOutput, error) {
		event, err := store.GetExecutionEvent(ctx, input.ID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("Santa event not found")
		}
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"get-santa-event",
				santaEventResource,
				err,
				"event_id",
				input.ID,
			)
		}
		return &santaEventGetOutput{Body: event}, nil
	})
}

func registerListSantaFileAccessEvents(api huma.API, store *events.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-file-access-events",
		Method:      http.MethodGet,
		Path:        santaFileAccessEventPath,
		Tags:        []string{santaEventsTag},
		Summary:     "List file access events",
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *santaFileAccessEventListInput) (*santaFileAccessEventListOutput, error) {
		rows, count, err := store.ListFileAccessEvents(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-santa-file-access-events", "Santa file access event", err)
		}
		return &santaFileAccessEventListOutput{
			Body: Page[events.FileAccessEvent]{Items: rows, Count: count},
		}, nil
	})
}

func registerGetSantaFileAccessEvent(api huma.API, store *events.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-file-access-event",
		Method:      http.MethodGet,
		Path:        santaFileAccessIDPath,
		Tags:        []string{santaEventsTag},
		Summary:     "Get a file access event",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *santaFileAccessEventGetInput) (*santaFileAccessEventGetOutput, error) {
		event, err := store.GetFileAccessEvent(ctx, input.ID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("Santa file access event not found")
		}
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"get-santa-file-access-event",
				"Santa file access event",
				err,
				"event_id", input.ID,
			)
		}
		return &santaFileAccessEventGetOutput{Body: event}, nil
	})
}
