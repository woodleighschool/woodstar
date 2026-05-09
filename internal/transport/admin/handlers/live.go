package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/models"
	queryinfra "github.com/woodleighschool/woodstar/internal/queries"
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

// RegisterLiveQueries registers the one-shot live query create endpoint. The
// matching SSE stream endpoint is wired directly on Chi (see routes.go).
func RegisterLiveQueries(
	api huma.API,
	manager *queryinfra.LiveQueryManager,
	db *database.DB,
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
		if manager == nil {
			return nil, huma.Error500InternalServerError("live queries are not configured")
		}
		hostIDs, err := input.Body.resolveTargets(ctx, db)
		if err != nil {
			return nil, err
		}
		handle := manager.Start(input.Body.SQL, hostIDs)
		return &liveQueryCreateOutput{Body: liveQueryHandleResponse(handle)}, nil
	})
}

func liveQueryHandleResponse(h *queryinfra.LiveQueryHandle) liveQueryHandleBody {
	return liveQueryHandleBody{
		ID:                h.ID,
		SQL:               h.SQL,
		StartedAt:         h.StartedAt,
		ResolvedHostCount: h.ResolvedHostCount,
	}
}

func (body liveQueryCreateBody) resolveTargets(ctx context.Context, db *database.DB) ([]int64, error) {
	if body.SQL == "" {
		return nil, huma.Error400BadRequest("sql is required")
	}
	hostIDs, err := parseIDList(body.Selected.Hosts, "selected.hosts")
	if err != nil {
		return nil, err
	}
	labelIDs, err := parseIDList(body.Selected.Labels, "selected.labels")
	if err != nil {
		return nil, err
	}
	scope := models.LabelScope{Mode: models.ScopeIncludeAny, LabelIDs: labelIDs}
	if len(scope.LabelIDs) > 0 && db != nil {
		matches, err := models.HostsMatchingScope(ctx, db, scope, "")
		if err != nil {
			return nil, err
		}
		hostIDs = mergePositiveIDs(hostIDs, matches)
	}
	if len(hostIDs) == 0 {
		return nil, huma.Error400BadRequest("no hosts targeted")
	}
	return hostIDs, nil
}

func mergePositiveIDs(a, b []int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	out := make([]int64, 0, len(a)+len(b))
	for _, id := range a {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range b {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// LiveQueryStreamHandler returns the SSE handler for /api/live-queries/{id}/stream.
// Auth must be applied by the caller via middleware.
func LiveQueryStreamHandler(manager *queryinfra.LiveQueryManager) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id, err := parseLiveQueryStreamID(req)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		streamLiveQuery(req.Context(), w, manager, id)
	}
}

var (
	errLiveStreamMissingID = errors.New("missing id")
	errLiveStreamInvalidID = errors.New("invalid id")
)

func parseLiveQueryStreamID(req *http.Request) (int64, error) {
	id := req.PathValue("id")
	if id == "" {
		return 0, errLiveStreamMissingID
	}
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, errLiveStreamInvalidID
	}
	return parsed, nil
}

func streamLiveQuery(
	ctx context.Context,
	w http.ResponseWriter,
	manager *queryinfra.LiveQueryManager,
	id int64,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	events, release, err := manager.Subscribe(id)
	if err != nil {
		writeSSE(w, "completed", map[string]string{"status": "completed"})
		flusher.Flush()
		return
	}
	defer release()
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !writeSSE(w, "ping", map[string]string{"status": "ok"}) {
				return
			}
			flusher.Flush()
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Status == "completed" {
				if !writeSSE(w, "completed", event) {
					return
				}
				flusher.Flush()
				return
			}
			if !writeSSE(w, "result", event) {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, event string, payload any) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	return err == nil
}
