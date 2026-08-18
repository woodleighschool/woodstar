package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
)

const (
	liveQueriesPath          = "/api/osquery/live-queries"
	liveQueryPollInterval    = time.Second
	liveQueryPingInterval    = 15 * time.Second
	liveQuerySnapshotTimeout = 2 * time.Second
)

type OsqueryLiveQueryCreateBody struct {
	SQL      string                       `json:"sql"`
	Selected OsqueryLiveQuerySelectedBody `json:"selected,omitzero"`
}

type OsqueryLiveQuerySelectedBody struct {
	Hosts  []int64 `json:"hosts,omitempty"`
	Labels []int64 `json:"labels,omitempty"`
}

type OsqueryLiveQueryTargetCountBody struct {
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

type liveQueryStreamKey struct{}

type liveQueryStreamState struct {
	ID        int64
	Snapshots []livequery.Snapshot
	Completed bool
}

type OsqueryLiveQuerySnapshotEvent struct {
	Type      string              `json:"type"       enum:"snapshot"`
	HostID    int64               `json:"host_id"`
	HostName  string              `json:"host_name"`
	Status    livequery.Status    `json:"status"     enum:"pending,collected,error,stopped"`
	Rows      []map[string]string `json:"rows"`
	Error     string              `json:"error,omitempty"`
	UpdatedAt time.Time           `json:"updated_at"`
}

func registerLiveQueries(
	humaAPI huma.API,
	streamingAPI huma.API,
	store *livequery.Store,
	hostStore *hosts.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-live-query",
		Method:        http.MethodPost,
		Path:          liveQueriesPath,
		Tags:          []string{api.TagOsqueryLiveQueries},
		Summary:       "Create a live query",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *liveQueryCreateInput) (*liveQueryCreateOutput, error) {
		targets, err := input.Body.resolveTargets(ctx, hostStore)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "create-live-query", err)
		}
		handle, err := store.Start(ctx, input.Body.SQL, targets)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "create-live-query", err)
		}
		activity.RecordUser(
			ctx,
			activityRecorder,
			logger,
			activity.AreaOsquery,
			activity.ActionLiveQueryStarted,
			activity.Resource("live_query", handle.ID, fmt.Sprintf("%d hosts", handle.ResolvedHostCount)),
		)
		return &liveQueryCreateOutput{Body: handle}, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID: "count-live-query-targets",
		Method:      http.MethodPost,
		Path:        liveQueriesPath + "/target-count",
		Tags:        []string{api.TagOsqueryLiveQueries},
		Summary:     "Count live query targets",
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *liveQueryTargetCountInput) (*liveQueryTargetCountOutput, error) {
		metrics, err := hostStore.CountSelectedTargets(ctx, input.Body.Selected.targetSelection(), time.Now().UTC())
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "count-live-query-targets", err)
		}
		return &liveQueryTargetCountOutput{Body: OsqueryLiveQueryTargetCountOutputBody{
			TargetsCount:   metrics.Total,
			TargetsOnline:  metrics.Online,
			TargetsOffline: metrics.Offline,
		}}, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID:   "delete-live-query",
		Method:        http.MethodDelete,
		Path:          liveQueriesPath + "/{id}",
		Tags:          []string{api.TagOsqueryLiveQueries},
		Summary:       "Delete a live query",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusNotFound},
	}, func(ctx context.Context, input *liveQueryInput) (*struct{}, error) {
		if err := store.Stop(ctx, input.ID); err != nil {
			if !errors.Is(err, livequery.ErrLiveQueryNotFound) {
				return nil, api.HandlerError(ctx, logger, "delete-live-query", err)
			}
			return nil, huma.Error404NotFound("live query not found")
		}
		activity.RecordUser(
			ctx,
			activityRecorder,
			logger,
			activity.AreaOsquery,
			activity.ActionLiveQueryStopped,
			activity.Resource("live_query", input.ID, "Live query"),
		)
		return &struct{}{}, nil
	})

	sse.Register(streamingAPI, huma.Operation{
		OperationID: "stream-live-query",
		Method:      http.MethodGet,
		Path:        liveQueriesPath + "/{id}/stream",
		Tags:        []string{api.TagOsqueryLiveQueries},
		Summary:     "Stream live query results",
		Errors:      []int{http.StatusNotFound},
		Middlewares: huma.Middlewares{loadLiveQuery(streamingAPI, store, logger)},
	}, map[string]any{
		"ping":      OsqueryLiveQueryPingEvent{},
		"snapshot":  OsqueryLiveQuerySnapshotEvent{},
		"completed": OsqueryLiveQueryCompletedEvent{},
	}, func(ctx context.Context, _ *liveQueryInput, send sse.Sender) {
		state, ok := ctx.Value(liveQueryStreamKey{}).(liveQueryStreamState)
		if !ok {
			return
		}
		streamLiveQuery(ctx, store, state, send, logger)
	})
	setLiveQueryStreamResponseSchema(streamingAPI)
}

// setLiveQueryStreamResponseSchema describes what the generated fetch client
// actually yields: each decoded SSE data payload, not Huma's documentation-only
// array of event envelopes.
func setLiveQueryStreamResponseSchema(humaAPI huma.API) {
	operation := humaAPI.OpenAPI().Paths[liveQueriesPath+"/{id}/stream"].Get
	operation.Responses["200"].Content["text/event-stream"].Schema = &huma.Schema{
		Title:       "Live query events",
		Description: "One payload per event.",
		OneOf: []*huma.Schema{
			humaAPI.OpenAPI().Components.Schemas.Schema(
				reflect.TypeFor[OsqueryLiveQueryPingEvent](),
				true,
				"ping",
			),
			humaAPI.OpenAPI().Components.Schemas.Schema(
				reflect.TypeFor[OsqueryLiveQuerySnapshotEvent](),
				true,
				"snapshot",
			),
			humaAPI.OpenAPI().Components.Schemas.Schema(
				reflect.TypeFor[OsqueryLiveQueryCompletedEvent](),
				true,
				"completed",
			),
		},
	}
}

func loadLiveQuery(
	humaAPI huma.API,
	store *livequery.Store,
	logger *slog.Logger,
) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if err != nil {
			next(ctx)
			return
		}
		snapshots, completed, err := readLiveQuerySnapshots(ctx.Context(), store, id)
		if err != nil {
			if !errors.Is(err, livequery.ErrLiveQueryNotFound) {
				logger.ErrorContext(ctx.Context(), "load live query stream failed",
					"operation", "stream-live-query", "live_query_id", id, "err", err)
				_ = huma.WriteErr(humaAPI, ctx, http.StatusInternalServerError, "request failed")
				return
			}
			_ = huma.WriteErr(humaAPI, ctx, http.StatusNotFound, "live query not found")
			return
		}
		next(huma.WithValue(ctx, liveQueryStreamKey{}, liveQueryStreamState{
			ID:        id,
			Snapshots: snapshots,
			Completed: completed,
		}))
	}
}

func readLiveQuerySnapshots(
	ctx context.Context,
	store *livequery.Store,
	id int64,
) ([]livequery.Snapshot, bool, error) {
	readCtx, cancel := context.WithTimeout(ctx, liveQuerySnapshotTimeout)
	defer cancel()
	return store.Snapshots(readCtx, id)
}

func (body OsqueryLiveQueryCreateBody) resolveTargets(
	ctx context.Context,
	hostStore *hosts.Store,
) ([]livequery.Target, error) {
	if strings.TrimSpace(body.SQL) == "" {
		return nil, huma.Error400BadRequest("sql is required")
	}
	resolved, err := hostStore.ResolveOnlineSelectedTargets(ctx, body.Selected.targetSelection(), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, huma.Error400BadRequest("no online hosts targeted")
	}
	targets := make([]livequery.Target, 0, len(resolved))
	for _, host := range resolved {
		targets = append(targets, livequery.Target{
			HostID:   host.ID,
			HostName: host.DisplayName,
		})
	}
	return targets, nil
}

func (body OsqueryLiveQuerySelectedBody) targetSelection() hosts.TargetSelection {
	return hosts.TargetSelection{HostIDs: body.Hosts, LabelIDs: body.Labels}
}

func streamLiveQuery(
	ctx context.Context,
	store *livequery.Store,
	state liveQueryStreamState,
	send sse.Sender,
	logger *slog.Logger,
) {
	seen := make(map[int64]livequery.Status, len(state.Snapshots))
	if !sendLiveQuerySnapshots(state.Snapshots, seen, send) {
		return
	}
	if state.Completed {
		_ = send.Data(OsqueryLiveQueryCompletedEvent{Type: "completed"})
		return
	}

	poll := time.NewTicker(liveQueryPollInterval)
	defer poll.Stop()
	ping := time.NewTicker(liveQueryPingInterval)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			if err := send.Data(OsqueryLiveQueryPingEvent{Type: "ping"}); err != nil {
				return
			}
		case <-poll.C:
			snapshots, completed, err := readLiveQuerySnapshots(ctx, store, state.ID)
			if err != nil {
				if ctx.Err() == nil {
					logger.WarnContext(ctx, "poll live query stream failed",
						"operation", "stream-live-query", "live_query_id", state.ID, "err", err)
				}
				return
			}
			if !sendLiveQuerySnapshots(snapshots, seen, send) {
				return
			}
			if completed {
				_ = send.Data(OsqueryLiveQueryCompletedEvent{Type: "completed"})
				return
			}
		}
	}
}

func sendLiveQuerySnapshots(
	snapshots []livequery.Snapshot,
	seen map[int64]livequery.Status,
	send sse.Sender,
) bool {
	for _, snapshot := range snapshots {
		if status, ok := seen[snapshot.HostID]; ok && status == snapshot.Status {
			continue
		}
		if err := send.Data(OsqueryLiveQuerySnapshotEventFromDomain(snapshot)); err != nil {
			return false
		}
		seen[snapshot.HostID] = snapshot.Status
	}
	return true
}

func OsqueryLiveQuerySnapshotEventFromDomain(
	snapshot livequery.Snapshot,
) OsqueryLiveQuerySnapshotEvent {
	return OsqueryLiveQuerySnapshotEvent{
		Type:      "snapshot",
		HostID:    snapshot.HostID,
		HostName:  snapshot.HostName,
		Status:    snapshot.Status,
		Rows:      snapshot.Rows,
		Error:     snapshot.Error,
		UpdatedAt: snapshot.UpdatedAt,
	}
}
