package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"

	"github.com/woodleighschool/woodstar/internal/agents/livequery"
	"github.com/woodleighschool/woodstar/internal/api/apihelpers"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

const liveQueriesTag = "Live Queries"

type targetResolver interface {
	ResolveSelectedTargets(context.Context, hosts.TargetSelection) ([]int64, error)
}

// liveQueryCreateBody mirrors the campaign body but is one-shot — no DB row,
// no detail page, no list page. Result events stream and disappear.
type liveQueryCreateBody struct {
	QueryID  *int64                `json:"query_id,omitempty"`
	SQL      string                `json:"sql"`
	Selected liveQuerySelectedBody `json:"selected,omitzero"`
}

type liveQuerySelectedBody struct {
	Hosts  []int64 `json:"hosts,omitempty"`
	Labels []int64 `json:"labels,omitempty"`
}

type liveQueryHandleBody struct {
	ID                int64     `json:"id"`
	SQL               string    `json:"sql"`
	StartedAt         time.Time `json:"started_at"`
	ResolvedHostCount int       `json:"resolved_host_count"`
}

type liveQueryCreateInput struct {
	Body liveQueryCreateBody
}

type liveQueryCreateOutput struct {
	Body liveQueryHandleBody
}

type liveQueryStreamInput struct {
	ID int64 `path:"id" minimum:"1"`
}

type liveQueryPingEvent struct {
	Status string `json:"status"`
}

type liveQueryCompletedEvent struct {
	Status string `json:"status"`
}

type liveQueryResultEvent livequery.Event

// RegisterLiveQueries registers the one-shot live query create endpoint. The
// matching SSE stream endpoint is registered through Huma's SSE support.
func RegisterLiveQueries(
	api huma.API,
	manager *livequery.Manager,
	resolver targetResolver,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-live-query",
		Method:        http.MethodPost,
		Path:          "/api/live-queries",
		Tags:          []string{liveQueriesTag},
		Summary:       "Start a live query against online hosts",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized},
	}, func(ctx context.Context, input *liveQueryCreateInput) (*liveQueryCreateOutput, error) {
		hostIDs, err := input.Body.resolveTargets(ctx, resolver)
		if err != nil {
			return nil, err
		}
		handle := manager.Start(input.Body.SQL, hostIDs)
		return &liveQueryCreateOutput{Body: liveQueryHandleResponse(handle)}, nil
	})

	sse.Register(api, huma.Operation{
		OperationID: "stream-live-query",
		Method:      http.MethodGet,
		Path:        "/api/live-queries/{id}/stream",
		Tags:        []string{liveQueriesTag},
		Summary:     "Stream live query results",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, map[string]any{
		"ping":      liveQueryPingEvent{},
		"result":    liveQueryResultEvent{},
		"completed": liveQueryCompletedEvent{},
	}, func(ctx context.Context, input *liveQueryStreamInput, send sse.Sender) {
		streamLiveQuery(ctx, manager, input.ID, send)
	})
}

func liveQueryHandleResponse(h livequery.Handle) liveQueryHandleBody {
	return liveQueryHandleBody{
		ID:                h.ID,
		SQL:               h.SQL,
		StartedAt:         h.StartedAt,
		ResolvedHostCount: h.ResolvedHostCount,
	}
}

func (body liveQueryCreateBody) resolveTargets(ctx context.Context, resolver targetResolver) ([]int64, error) {
	if body.SQL == "" {
		return nil, huma.Error400BadRequest("sql is required")
	}
	hostIDs, err := apihelpers.ParseIDList(body.Selected.Hosts, "selected.hosts")
	if err != nil {
		return nil, err
	}
	labelIDs, err := apihelpers.ParseIDList(body.Selected.Labels, "selected.labels")
	if err != nil {
		return nil, err
	}
	resolved, err := resolver.ResolveSelectedTargets(ctx, hosts.TargetSelection{
		HostIDs:  hostIDs,
		LabelIDs: labelIDs,
	})
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, huma.Error400BadRequest("no hosts targeted")
	}
	return resolved, nil
}

func streamLiveQuery(
	ctx context.Context,
	manager *livequery.Manager,
	id int64,
	send sse.Sender,
) {
	events, release, err := manager.Subscribe(id)
	if err != nil {
		_ = send.Data(liveQueryCompletedEvent{Status: "completed"})
		return
	}
	defer release()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := send.Data(liveQueryPingEvent{Status: "ok"}); err != nil {
				return
			}
		case event, ok := <-events:
			if !ok {
				_ = send.Data(liveQueryCompletedEvent{Status: "completed"})
				return
			}
			if event.Status == "completed" {
				_ = send.Data(liveQueryCompletedEvent{Status: "completed"})
				return
			}
			if err := send.Data(liveQueryResultEvent(event)); err != nil {
				return
			}
		}
	}
}
