package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
)

const liveQueriesTag = "Live Queries"

type liveQueryCreateBody struct {
	ReportID *int64                `json:"report_id,omitempty"`
	SQL      string                `json:"sql"`
	Selected liveQuerySelectedBody `json:"selected,omitzero"`
}

type liveQuerySelectedBody struct {
	Hosts  []int64 `json:"hosts,omitempty"`
	Labels []int64 `json:"labels,omitempty"`
}

type liveQueryTargetCountBody struct {
	ReportID *int64                `json:"report_id,omitempty"`
	Selected liveQuerySelectedBody `json:"selected,omitzero"`
}

type liveQueryTargetCountOutputBody struct {
	TargetsCount   int `json:"targets_count"`
	TargetsOnline  int `json:"targets_online"`
	TargetsOffline int `json:"targets_offline"`
}

type liveQueryCreateInput struct {
	Body liveQueryCreateBody
}

type liveQueryCreateOutput struct {
	Body livequery.Handle
}

type liveQueryTargetCountInput struct {
	Body liveQueryTargetCountBody
}

type liveQueryTargetCountOutput struct {
	Body liveQueryTargetCountOutputBody
}

type liveQueryStreamInput struct {
	ID int64 `path:"id"`
}

type liveQueryStopInput struct {
	ID int64 `path:"id"`
}

type liveQueryStopOutput struct{}

type liveQueryPingEvent struct {
	Status string `json:"status"`
}

type liveQueryCompletedEvent struct {
	Status string `json:"status"`
}

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
		return &liveQueryCreateOutput{Body: manager.Start(input.Body.SQL, hostIDs)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "count-live-query-targets",
		Method:      http.MethodPost,
		Path:        "/api/live-queries/targets/count",
		Tags:        []string{liveQueriesTag},
		Summary:     "Count live query targets",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized},
	}, func(ctx context.Context, input *liveQueryTargetCountInput) (*liveQueryTargetCountOutput, error) {
		metrics, err := hostStore.CountSelectedTargets(ctx, input.Body.Selected.targetSelection(), time.Now().UTC())
		if err != nil {
			return nil, err
		}
		return &liveQueryTargetCountOutput{Body: liveQueryTargetCountOutputBody{
			TargetsCount:   metrics.Total,
			TargetsOnline:  metrics.Online,
			TargetsOffline: metrics.Offline,
		}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "stop-live-query",
		Method:        http.MethodPost,
		Path:          "/api/live-queries/{id}/stop",
		Tags:          []string{liveQueriesTag},
		Summary:       "Stop a running live query",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(_ context.Context, input *liveQueryStopInput) (*liveQueryStopOutput, error) {
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
		"result":    livequery.Event{},
		"completed": liveQueryCompletedEvent{},
	}, func(ctx context.Context, input *liveQueryStreamInput, send sse.Sender) {
		streamLiveQuery(ctx, manager, input.ID, send)
	})
}

func (body liveQueryCreateBody) resolveTargets(ctx context.Context, hostStore *hosts.Store) ([]int64, error) {
	if body.SQL == "" {
		return nil, huma.Error400BadRequest("sql is required")
	}
	resolved, err := hostStore.ResolveOnlineSelectedTargets(ctx, body.Selected.targetSelection(), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, huma.Error400BadRequest("no online hosts targeted")
	}
	return resolved, nil
}

func (body liveQuerySelectedBody) targetSelection() hosts.TargetSelection {
	return hosts.TargetSelection{HostIDs: body.Hosts, LabelIDs: body.Labels}
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
			if err := send.Data(event); err != nil {
				return
			}
		}
	}
}
