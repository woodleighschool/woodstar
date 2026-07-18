package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
)

const (
	liveQueriesTag  = "Live Queries"
	liveQueriesPath = "/api/live-queries"
)

type OsqueryLiveQueryCreateBody struct {
	ReportID *int64                       `json:"report_id,omitempty"`
	SQL      string                       `json:"sql"`
	Selected OsqueryLiveQuerySelectedBody `json:"selected,omitzero"`
}

type OsqueryLiveQuerySelectedBody struct {
	Hosts  []int64 `json:"hosts,omitempty"`
	Labels []int64 `json:"labels,omitempty"`
}

type OsqueryLiveQueryTargetCountBody struct {
	ReportID *int64                       `json:"report_id,omitempty"`
	Selected OsqueryLiveQuerySelectedBody `json:"selected,omitzero"`
}

type OsqueryLiveQueryTargetCountOutputBody struct {
	TargetsCount   int32 `json:"targets_count"`
	TargetsOnline  int32 `json:"targets_online"`
	TargetsOffline int32 `json:"targets_offline"`
}

type liveQueryCreateInput struct {
	Body OsqueryLiveQueryCreateBody
}

type liveQueryCreateOutput struct {
	Body livequery.Handle
}

type liveQueryTargetCountInput struct {
	Body OsqueryLiveQueryTargetCountBody
}

type liveQueryTargetCountOutput struct {
	Body OsqueryLiveQueryTargetCountOutputBody
}

type liveQueryInput struct {
	ID int64 `path:"id"`
}

type OsqueryLiveQueryPingEvent struct {
	Type string `json:"type" enum:"ping"`
}

type OsqueryLiveQueryCompletedEvent struct {
	Type string `json:"type" enum:"completed"`
}

type liveQuerySubscriptionKey struct{}

type OsqueryLiveQueryResultEvent struct {
	Type     string          `json:"type"                enum:"result"`
	HostID   int64           `json:"host_id,omitempty"`
	HostName string          `json:"host_name,omitempty"`
	Status   string          `json:"status"              enum:"success,error,stopped,overflow"`
	Data     json.RawMessage `json:"data,omitempty"`
	Error    string          `json:"error,omitempty"`
}

func registerLiveQueries(
	api huma.API,
	streamingAPI huma.API,
	manager *livequery.Manager,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-live-query",
		Method:        http.MethodPost,
		Path:          liveQueriesPath,
		Tags:          []string{liveQueriesTag},
		Summary:       "Start a live run against online hosts",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *liveQueryCreateInput) (*liveQueryCreateOutput, error) {
		hostIDs, err := input.Body.resolveTargets(ctx, hostStore)
		if err != nil {
			return nil, handlerError(ctx, logger, "create-live-query", err)
		}
		return &liveQueryCreateOutput{Body: manager.Start(input.Body.SQL, hostIDs)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "count-live-query-targets",
		Method:      http.MethodPost,
		Path:        liveQueriesPath + "/targets/count",
		Tags:        []string{liveQueriesTag},
		Summary:     "Count live query targets",
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *liveQueryTargetCountInput) (*liveQueryTargetCountOutput, error) {
		metrics, err := hostStore.CountSelectedTargets(ctx, input.Body.Selected.targetSelection(), time.Now().UTC())
		if err != nil {
			return nil, handlerError(ctx, logger, "count-live-query-targets", err)
		}
		return &liveQueryTargetCountOutput{Body: OsqueryLiveQueryTargetCountOutputBody{
			TargetsCount:   metrics.Total,
			TargetsOnline:  metrics.Online,
			TargetsOffline: metrics.Offline,
		}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-live-query",
		Method:        http.MethodDelete,
		Path:          liveQueriesPath + "/{id}",
		Tags:          []string{liveQueriesTag},
		Summary:       "Stop a running live query",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusNotFound},
	}, func(_ context.Context, input *liveQueryInput) (*struct{}, error) {
		if err := manager.Stop(input.ID); err != nil {
			return nil, huma.Error404NotFound("live query not found")
		}
		return &struct{}{}, nil
	})

	sse.Register(streamingAPI, huma.Operation{
		OperationID: "stream-live-query",
		Method:      http.MethodGet,
		Path:        liveQueriesPath + "/{id}/stream",
		Tags:        []string{liveQueriesTag},
		Summary:     "Stream live query results",
		Errors:      []int{http.StatusNotFound},
		Middlewares: huma.Middlewares{subscribeLiveQuery(streamingAPI, manager)},
	}, map[string]any{
		"ping":      OsqueryLiveQueryPingEvent{},
		"result":    OsqueryLiveQueryResultEvent{},
		"completed": OsqueryLiveQueryCompletedEvent{},
	}, func(ctx context.Context, _ *liveQueryInput, send sse.Sender) {
		events, ok := ctx.Value(liveQuerySubscriptionKey{}).(<-chan livequery.Event)
		if !ok {
			return
		}
		streamLiveQuery(ctx, events, send)
	})
	setLiveQueryStreamResponseSchema(streamingAPI)
}

// setLiveQueryStreamResponseSchema describes what the generated fetch client
// actually yields: each decoded SSE data payload, not Huma's documentation-only
// array of event envelopes.
func setLiveQueryStreamResponseSchema(api huma.API) {
	operation := api.OpenAPI().Paths[liveQueriesPath+"/{id}/stream"].Get
	operation.Responses["200"].Content["text/event-stream"].Schema = &huma.Schema{
		Title:       "Live query events",
		Description: "One decoded live-query payload per server-sent event.",
		OneOf: []*huma.Schema{
			api.OpenAPI().Components.Schemas.Schema(
				reflect.TypeFor[OsqueryLiveQueryPingEvent](),
				true,
				"ping",
			),
			api.OpenAPI().Components.Schemas.Schema(
				reflect.TypeFor[OsqueryLiveQueryResultEvent](),
				true,
				"result",
			),
			api.OpenAPI().Components.Schemas.Schema(
				reflect.TypeFor[OsqueryLiveQueryCompletedEvent](),
				true,
				"completed",
			),
		},
	}
}

func subscribeLiveQuery(api huma.API, manager *livequery.Manager) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if err != nil {
			next(ctx)
			return
		}
		events, release, err := manager.Subscribe(id)
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusNotFound, "live query not found")
			return
		}
		defer release()
		next(huma.WithValue(ctx, liveQuerySubscriptionKey{}, events))
	}
}

func (body OsqueryLiveQueryCreateBody) resolveTargets(ctx context.Context, hostStore *hosts.Store) ([]int64, error) {
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

func (body OsqueryLiveQuerySelectedBody) targetSelection() hosts.TargetSelection {
	return hosts.TargetSelection{HostIDs: body.Hosts, LabelIDs: body.Labels}
}

func streamLiveQuery(
	ctx context.Context,
	events <-chan livequery.Event,
	send sse.Sender,
) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := send.Data(OsqueryLiveQueryPingEvent{Type: "ping"}); err != nil {
				return
			}
		case event, ok := <-events:
			if !ok {
				_ = send.Data(OsqueryLiveQueryCompletedEvent{Type: "completed"})
				return
			}
			if err := send.Data(OsqueryLiveQueryResultEventFromDomain(event)); err != nil {
				return
			}
		}
	}
}

func OsqueryLiveQueryResultEventFromDomain(event livequery.Event) OsqueryLiveQueryResultEvent {
	return OsqueryLiveQueryResultEvent{
		Type:     "result",
		HostID:   event.HostID,
		HostName: event.HostName,
		Status:   string(event.Status),
		Data:     event.Data,
		Error:    event.Error,
	}
}
