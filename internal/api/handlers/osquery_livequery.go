package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/humaschema"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
)

const liveQueriesTag = "Live Queries"

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

type liveQueryStreamInput struct {
	ID int64 `path:"id"`
}

type liveQueryStopInput struct {
	ID int64 `path:"id"`
}

type liveQueryStopOutput struct{}

type liveQueryPingStatus string

const liveQueryPingStatusOK liveQueryPingStatus = "ok"

var liveQueryPingStatusValues = []liveQueryPingStatus{liveQueryPingStatusOK}

type OsqueryLiveQueryPingEvent struct {
	Status liveQueryPingStatus `json:"status"`
}

type liveQueryCompletedStatus string

const liveQueryCompletedStatusCompleted liveQueryCompletedStatus = "completed"

var liveQueryCompletedStatusValues = []liveQueryCompletedStatus{liveQueryCompletedStatusCompleted}

type OsqueryLiveQueryCompletedEvent struct {
	Status liveQueryCompletedStatus `json:"status"`
}

type liveQueryResultStatus string

const (
	liveQueryResultStatusSuccess liveQueryResultStatus = "success"
	liveQueryResultStatusError   liveQueryResultStatus = "error"
	liveQueryResultStatusStopped liveQueryResultStatus = "stopped"
)

var liveQueryResultStatusValues = []liveQueryResultStatus{
	liveQueryResultStatusSuccess,
	liveQueryResultStatusError,
	liveQueryResultStatusStopped,
}

type OsqueryLiveQueryResultEvent struct {
	HostID   int64                 `json:"host_id,omitempty"`
	HostName string                `json:"host_name,omitempty"`
	Status   liveQueryResultStatus `json:"status"`
	Data     json.RawMessage       `json:"data,omitempty"`
	Error    string                `json:"error,omitempty"`
}

func (liveQueryPingStatus) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(liveQueryPingStatusValues...)
}

func (liveQueryCompletedStatus) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(liveQueryCompletedStatusValues...)
}

func (liveQueryResultStatus) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(liveQueryResultStatusValues...)
}

func registerLiveQueries(
	api huma.API,
	manager *livequery.Manager,
	hostStore *hosts.Store,
	logger *slog.Logger,
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
			return nil, handlerError(ctx, logger, "create-live-query", err)
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
			return nil, handlerError(ctx, logger, "count-live-query-targets", err)
		}
		return &liveQueryTargetCountOutput{Body: OsqueryLiveQueryTargetCountOutputBody{
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
		"ping":      OsqueryLiveQueryPingEvent{},
		"result":    OsqueryLiveQueryResultEvent{},
		"completed": OsqueryLiveQueryCompletedEvent{},
	}, func(ctx context.Context, input *liveQueryStreamInput, send sse.Sender) {
		streamLiveQuery(ctx, manager, input.ID, send)
	})
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
	manager *livequery.Manager,
	id int64,
	send sse.Sender,
) {
	events, release, err := manager.Subscribe(id)
	if err != nil {
		_ = send.Data(OsqueryLiveQueryCompletedEvent{Status: liveQueryCompletedStatusCompleted})
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
			if err := send.Data(OsqueryLiveQueryPingEvent{Status: liveQueryPingStatusOK}); err != nil {
				return
			}
		case event, ok := <-events:
			if !ok {
				_ = send.Data(OsqueryLiveQueryCompletedEvent{Status: liveQueryCompletedStatusCompleted})
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
		HostID:   event.HostID,
		HostName: event.HostName,
		Status:   liveQueryResultStatus(event.Status),
		Data:     event.Data,
		Error:    event.Error,
	}
}
