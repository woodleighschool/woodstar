//nolint:dupl // Huma route registration is intentionally explicit per resource.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/transport/admin/adminctx"
)

const (
	queriesTag    = "Queries"
	queryResource = "query"
	queryIDPath   = "/api/queries/{id}"
)

type labelScopeBody struct {
	Mode     models.LabelScopeMode `json:"mode,omitempty"      enum:"include_any,include_all,exclude_any"`
	LabelIDs []string              `json:"label_ids,omitempty"`
}

type queryBody struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	Query             string         `json:"query"`
	Platform          *string        `json:"platform,omitempty"`
	MinOsqueryVersion *string        `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int            `json:"schedule_interval"`
	LabelScope        labelScopeBody `json:"label_scope,omitzero"`
	CreatedByUserID   *string        `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// queryMutationBody is the POST body shape (mutable fields only).
type queryMutationBody struct {
	Name              string         `json:"name"`
	Description       string         `json:"description,omitempty"`
	Query             string         `json:"query"`
	Platform          *string        `json:"platform,omitempty"`
	MinOsqueryVersion *string        `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int            `json:"schedule_interval,omitempty"`
	LabelScope        labelScopeBody `json:"label_scope"`
}

// queryPutBody mirrors queryBody so the autopatch round-trip (GET → merge →
// PUT) accepts the response shape verbatim. Read-only fields are accepted
// but ignored.
type queryPutBody struct {
	ID                string         `json:"id,omitempty"`
	Name              string         `json:"name"`
	Description       string         `json:"description,omitempty"`
	Query             string         `json:"query"`
	Platform          *string        `json:"platform,omitempty"`
	MinOsqueryVersion *string        `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int            `json:"schedule_interval,omitempty"`
	LabelScope        labelScopeBody `json:"label_scope"`
	CreatedByUserID   *string        `json:"created_by_user_id,omitempty"`
	CreatedAt         *time.Time     `json:"created_at,omitempty"`
	UpdatedAt         *time.Time     `json:"updated_at,omitempty"`
}

type queryListInput struct {
	Q              string `query:"q,omitempty"`
	Platform       string `query:"platform,omitempty"`
	Page           int    `query:"page,omitempty"`
	PerPage        int    `query:"per_page,omitempty"`
	OrderKey       string `query:"order_key,omitempty"`
	OrderDirection string `query:"order_direction,omitempty"`
}

type queryGetInput struct {
	ID string `path:"id"`
}

type queryCreateInput struct {
	Body queryMutationBody
}

type queryPutInput struct {
	ID   string `path:"id"`
	Body queryPutBody
}

type queryDeleteInput struct {
	ID string `path:"id"`
}

type queryListOutput struct {
	Body struct {
		Items []queryBody `json:"items"`
		Count int         `json:"count"`
	}
}

type queryOutput struct {
	Body queryBody
}

type queryResultsOutput struct {
	Body struct {
		Items []queryResultBody `json:"items"`
	}
}

type queryResultBody struct {
	QueryID     string            `json:"query_id"`
	QueryName   string            `json:"query_name"`
	HostID      string            `json:"host_id"`
	HostName    string            `json:"host_name"`
	Columns     map[string]string `json:"columns"`
	LastFetched *time.Time        `json:"last_fetched,omitempty"`
}

type hostReportsOutput struct {
	Body struct {
		Items []hostReportBody `json:"items"`
	}
}

type hostReportBody struct {
	ReportID        string            `json:"report_id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	LastFetched     *time.Time        `json:"last_fetched,omitempty"`
	FirstResult     map[string]string `json:"first_result,omitempty"`
	HostResultCount int               `json:"n_host_results"`
}

type hostQueryResultsOutput struct {
	Body struct {
		QueryID     string            `json:"query_id"`
		HostID      string            `json:"host_id"`
		HostName    string            `json:"host_name"`
		LastFetched *time.Time        `json:"last_fetched,omitempty"`
		Items       []queryResultBody `json:"items"`
	}
}

type hostQueryResultsInput struct {
	ID      string `path:"id"`
	QueryID string `path:"query_id"`
}

// RegisterQueries registers saved-query and report endpoints.
func RegisterQueries(api huma.API, store *models.QueryStore, hosts *models.HostStore) {
	registerListQueries(api, store)
	registerCreateQuery(api, store)
	registerGetQuery(api, store)
	registerUpdateQuery(api, store)
	registerDeleteQuery(api, store)
	registerQueryResults(api, store)
	registerHostQueries(api, store, hosts)
	registerHostQueryResults(api, store, hosts)
}

func registerListQueries(api huma.API, store *models.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-queries",
		Method:      http.MethodGet,
		Path:        "/api/queries",
		Tags:        []string{queriesTag},
		Summary:     "List saved queries",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *queryListInput) (*queryListOutput, error) {
		items, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, err
		}
		out := &queryListOutput{}
		out.Body.Items = make([]queryBody, 0, len(items))
		out.Body.Count = count
		for i := range items {
			out.Body.Items = append(out.Body.Items, queryResponse(&items[i]))
		}
		return out, nil
	})
}

func registerCreateQuery(api huma.API, store *models.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-query",
		Method:        http.MethodPost,
		Path:          "/api/queries",
		Tags:          []string{queriesTag},
		Summary:       "Create a saved query",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict},
	}, func(ctx context.Context, input *queryCreateInput) (*queryOutput, error) {
		params, err := input.Body.createParams(currentUserID(ctx))
		if err != nil {
			return nil, err
		}
		query, err := store.Create(ctx, params)
		if err != nil {
			return nil, resourceMutationError(queryResource, err)
		}
		return &queryOutput{Body: queryResponse(query)}, nil
	})
}

func registerGetQuery(api huma.API, store *models.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "get-query",
		Method:      http.MethodGet,
		Path:        queryIDPath,
		Tags:        []string{queriesTag},
		Summary:     "Get a saved query",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *queryGetInput) (*queryOutput, error) {
		id, err := parseResourceID(input.ID, queryResource)
		if err != nil {
			return nil, err
		}
		query, err := store.GetByID(ctx, id)
		if err != nil {
			return nil, resourceMutationError(queryResource, err)
		}
		return &queryOutput{Body: queryResponse(query)}, nil
	})
}

func registerUpdateQuery(api huma.API, store *models.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "put-query",
		Method:      http.MethodPut,
		Path:        queryIDPath,
		Tags:        []string{queriesTag},
		Summary:     "Replace a saved query",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *queryPutInput) (*queryOutput, error) {
		id, err := parseResourceID(input.ID, queryResource)
		if err != nil {
			return nil, err
		}
		params, err := input.Body.updateParams()
		if err != nil {
			return nil, err
		}
		query, err := store.Update(ctx, id, params)
		if err != nil {
			return nil, resourceMutationError(queryResource, err)
		}
		return &queryOutput{Body: queryResponse(query)}, nil
	})
}

func registerDeleteQuery(api huma.API, store *models.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-query",
		Method:      http.MethodDelete,
		Path:        queryIDPath,
		Tags:        []string{queriesTag},
		Summary:     "Delete a saved query",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *queryDeleteInput) (*struct{}, error) {
		id, err := parseResourceID(input.ID, queryResource)
		if err != nil {
			return nil, err
		}
		if err := store.Delete(ctx, id); err != nil {
			return nil, resourceMutationError(queryResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerQueryResults(api huma.API, store *models.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-query-results",
		Method:      http.MethodGet,
		Path:        "/api/queries/{id}/results",
		Tags:        []string{queriesTag},
		Summary:     "List latest report snapshots for a query",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *queryGetInput) (*queryResultsOutput, error) {
		id, err := parseResourceID(input.ID, queryResource)
		if err != nil {
			return nil, err
		}
		rows, err := store.Results(ctx, id)
		if err != nil {
			return nil, err
		}
		out := &queryResultsOutput{}
		out.Body.Items = queryResultResponses(rows)
		return out, nil
	})
}

func registerHostQueries(api huma.API, store *models.QueryStore, hosts *models.HostStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-queries",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/queries",
		Tags:        []string{queriesTag, hostsTag},
		Summary:     "List reports for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostReportsOutput, error) {
		id, err := parseHostID(input.ID)
		if err != nil {
			return nil, err
		}
		host, err := hosts.GetByID(ctx, id)
		if errors.Is(err, models.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		rows, err := store.HostReports(ctx, *host)
		if err != nil {
			return nil, err
		}
		out := &hostReportsOutput{}
		out.Body.Items = hostReportResponses(rows)
		return out, nil
	})
}

func registerHostQueryResults(api huma.API, store *models.QueryStore, hosts *models.HostStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-query-results",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/queries/{query_id}",
		Tags:        []string{queriesTag, hostsTag},
		Summary:     "List report rows for one host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostQueryResultsInput) (*hostQueryResultsOutput, error) {
		hostID, err := parseHostID(input.ID)
		if err != nil {
			return nil, err
		}
		queryID, err := parseResourceID(input.QueryID, queryResource)
		if err != nil {
			return nil, err
		}
		host, err := hosts.GetByID(ctx, hostID)
		if errors.Is(err, models.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		rows, lastFetched, err := store.HostQueryResults(ctx, hostID, queryID)
		if err != nil {
			return nil, err
		}
		out := &hostQueryResultsOutput{}
		out.Body.QueryID = input.QueryID
		out.Body.HostID = input.ID
		out.Body.HostName = host.DisplayName
		out.Body.LastFetched = lastFetched
		out.Body.Items = queryResultResponses(rows)
		return out, nil
	})
}

func (input queryListInput) params() models.QueryListParams {
	return models.QueryListParams{
		ListParams: models.CleanListParams(models.ListParams{
			Q:              input.Q,
			Page:           input.Page,
			PerPage:        input.PerPage,
			OrderKey:       input.OrderKey,
			OrderDirection: input.OrderDirection,
		}),
		Platform: strings.TrimSpace(input.Platform),
	}
}

func (body queryMutationBody) createParams(userID *int64) (models.QueryCreate, error) {
	scope, err := body.LabelScope.model()
	if err != nil {
		return models.QueryCreate{}, err
	}
	return models.QueryCreate{
		Name:              body.Name,
		Description:       body.Description,
		Query:             body.Query,
		Platform:          body.Platform,
		MinOsqueryVersion: body.MinOsqueryVersion,
		ScheduleInterval:  body.ScheduleInterval,
		LoggingType:       models.QueryLoggingSnapshot,
		LabelScope:        scope,
		CreatedByUserID:   userID,
	}, nil
}

func (body queryPutBody) updateParams() (models.QueryUpdate, error) {
	scope, err := body.LabelScope.model()
	if err != nil {
		return models.QueryUpdate{}, err
	}
	return models.QueryUpdate{
		Name:              body.Name,
		Description:       body.Description,
		Query:             body.Query,
		Platform:          body.Platform,
		MinOsqueryVersion: body.MinOsqueryVersion,
		ScheduleInterval:  body.ScheduleInterval,
		LoggingType:       models.QueryLoggingSnapshot,
		LabelScope:        scope,
	}, nil
}

func queryResponse(query *models.Query) queryBody {
	return queryBody{
		ID:                strconv.FormatInt(query.ID, 10),
		Name:              query.Name,
		Description:       query.Description,
		Query:             query.Query,
		Platform:          query.Platform,
		MinOsqueryVersion: query.MinOsqueryVersion,
		ScheduleInterval:  query.ScheduleInterval,
		LabelScope:        labelScopeResponse(query.LabelScope),
		CreatedByUserID:   idStringPtr(query.CreatedByUserID),
		CreatedAt:         query.CreatedAt,
		UpdatedAt:         query.UpdatedAt,
	}
}

func queryResultResponses(rows []models.QueryResult) []queryResultBody {
	out := make([]queryResultBody, 0, len(rows))
	for _, row := range rows {
		var lastFetched *time.Time
		if !row.LastFetched.IsZero() {
			lastFetched = &row.LastFetched
		}
		out = append(out, queryResultBody{
			QueryID:     strconv.FormatInt(row.QueryID, 10),
			QueryName:   row.QueryName,
			HostID:      strconv.FormatInt(row.HostID, 10),
			HostName:    row.HostName,
			Columns:     row.Columns,
			LastFetched: lastFetched,
		})
	}
	return out
}

func hostReportResponses(rows []models.HostReport) []hostReportBody {
	out := make([]hostReportBody, 0, len(rows))
	for _, row := range rows {
		out = append(out, hostReportBody{
			ReportID:        strconv.FormatInt(row.ReportID, 10),
			Name:            row.Name,
			Description:     row.Description,
			LastFetched:     row.LastFetched,
			FirstResult:     row.FirstResult,
			HostResultCount: row.HostResultCount,
		})
	}
	return out
}

func (body labelScopeBody) model() (models.LabelScope, error) {
	ids, err := parseIDList(body.LabelIDs, "label_ids")
	if err != nil {
		return models.LabelScope{}, err
	}
	return models.NormalizeLabelScope(models.LabelScope{Mode: body.Mode, LabelIDs: ids}), nil
}

func labelScopeResponse(scope models.LabelScope) labelScopeBody {
	ids := make([]string, 0, len(scope.LabelIDs))
	for _, id := range scope.LabelIDs {
		ids = append(ids, strconv.FormatInt(id, 10))
	}
	return labelScopeBody{Mode: scope.Mode, LabelIDs: ids}
}

func currentUserID(ctx context.Context) *int64 {
	user, ok := adminctx.UserFromContext(ctx)
	if !ok {
		return nil
	}
	return &user.ID
}
