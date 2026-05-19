package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"

	"github.com/woodleighschool/woodstar/internal/agents/livequery"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

const liveQueriesTag = "Live Queries"

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

type liveQueryTargetCountBody struct {
	QueryID  *int64                `json:"query_id,omitempty"`
	Selected liveQuerySelectedBody `json:"selected,omitzero"`
}

type liveQueryTargetCountOutputBody struct {
	TargetsCount           int `json:"targets_count"`
	TargetsOnline          int `json:"targets_online"`
	TargetsOffline         int `json:"targets_offline"`
	TargetsMissingInAction int `json:"targets_missing_in_action"`
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

type liveQueryTargetCountInput struct {
	Body liveQueryTargetCountBody
}

type liveQueryTargetCountOutput struct {
	Body liveQueryTargetCountOutputBody
}

type liveQueryStreamInput struct {
	ID int64 `path:"id" minimum:"1"`
}

type liveQueryStopInput struct {
	ID int64 `path:"id" minimum:"1"`
}

type liveQueryStopOutput struct{}

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
	hostStore *hosts.Store,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-live-query",
		Method:        http.MethodPost,
		Path:          "/api/live-queries",
		Tags:          []string{liveQueriesTag},
		Summary:       "Start a live run against online hosts",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized},
	}, func(ctx context.Context, input *liveQueryCreateInput) (*liveQueryCreateOutput, error) {
		hostIDs, err := input.Body.resolveTargets(ctx, hostStore)
		if err != nil {
			return nil, err
		}
		handle := manager.Start(input.Body.SQL, hostIDs)
		return &liveQueryCreateOutput{Body: liveQueryHandleResponse(handle)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "count-live-query-targets",
		Method:      http.MethodPost,
		Path:        "/api/live-queries/targets/count",
		Tags:        []string{liveQueriesTag},
		Summary:     "Count live query targets",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized},
	}, func(ctx context.Context, input *liveQueryTargetCountInput) (*liveQueryTargetCountOutput, error) {
		selection, err := input.Body.Selected.targetSelection()
		if err != nil {
			return nil, err
		}
		metrics, err := hostStore.CountSelectedTargets(ctx, selection, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		return &liveQueryTargetCountOutput{Body: liveQueryTargetCountResponse(metrics)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "stop-live-query",
		Method:        http.MethodPost,
		Path:          "/api/live-queries/{id}/stop",
		Tags:          []string{liveQueriesTag},
		Summary:       "Stop a running live query",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *liveQueryStopInput) (*liveQueryStopOutput, error) {
		if err := manager.Stop(input.ID); err != nil {
			return nil, huma.Error404NotFound("live query not found")
		}
		return &liveQueryStopOutput{}, nil
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

func liveQueryTargetCountResponse(metrics hosts.TargetMetrics) liveQueryTargetCountOutputBody {
	return liveQueryTargetCountOutputBody{
		TargetsCount:           metrics.Total,
		TargetsOnline:          metrics.Online,
		TargetsOffline:         metrics.Offline,
		TargetsMissingInAction: metrics.MissingInAction,
	}
}

func (body liveQueryCreateBody) resolveTargets(ctx context.Context, hostStore *hosts.Store) ([]int64, error) {
	if body.SQL == "" {
		return nil, huma.Error400BadRequest("sql is required")
	}
	selection, err := body.Selected.targetSelection()
	if err != nil {
		return nil, err
	}
	resolved, err := hostStore.ResolveOnlineSelectedTargets(ctx, selection, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, huma.Error400BadRequest("no online hosts targeted")
	}
	return resolved, nil
}

func (body liveQuerySelectedBody) targetSelection() (hosts.TargetSelection, error) {
	hostIDs, err := parseIDList(body.Hosts, "selected.hosts")
	if err != nil {
		return hosts.TargetSelection{}, err
	}
	labelIDs, err := parseIDList(body.Labels, "selected.labels")
	if err != nil {
		return hosts.TargetSelection{}, err
	}
	return hosts.TargetSelection{HostIDs: hostIDs, LabelIDs: labelIDs}, nil
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
