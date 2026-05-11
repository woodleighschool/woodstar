//nolint:dupl // Huma route registration is intentionally explicit per resource.
package agentapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/agents/queries"
	"github.com/woodleighschool/woodstar/internal/api/apihelpers"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

const (
	queriesTag    = "Queries"
	queryResource = "query"
	queryIDPath   = "/api/queries/{id}"
)

type queryBody struct {
	ID                int64          `json:"id"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	Query             string         `json:"query"`
	Platform          *string        `json:"platform,omitempty"`
	MinOsqueryVersion *string        `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int            `json:"schedule_interval"`
	LabelScope        labelScopeBody `json:"label_scope,omitzero"`
	CreatedByUserID   *int64         `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type queryMutationBody struct {
	Name              string         `json:"name"`
	Description       string         `json:"description,omitempty"`
	Query             string         `json:"query"`
	Platform          *string        `json:"platform,omitempty"`
	MinOsqueryVersion *string        `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int            `json:"schedule_interval,omitempty"`
	LabelScope        labelScopeBody `json:"label_scope"`
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
	Body queryMutationBody
}

type queryDeleteInput struct {
	ID string `path:"id"`
}

type queryBulkDeleteInput struct {
	Body bulkIDsBody
}

func (i queryBulkDeleteInput) ids() ([]int64, error) {
	return apihelpers.CleanBulkIDs(i.Body.IDs, "query IDs")
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
	QueryID     int64             `json:"query_id"`
	QueryName   string            `json:"query_name"`
	HostID      int64             `json:"host_id"`
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
	ReportID        int64             `json:"report_id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	LastFetched     *time.Time        `json:"last_fetched,omitempty"`
	FirstResult     map[string]string `json:"first_result,omitempty"`
	HostResultCount int               `json:"n_host_results"`
}

type hostQueryResultsOutput struct {
	Body struct {
		QueryID     int64             `json:"query_id"`
		HostID      int64             `json:"host_id"`
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
func RegisterQueries(api huma.API, queryStore *queries.QueryStore, hostStore *hosts.HostStore) {
	registerListQueries(api, queryStore)
	registerCreateQuery(api, queryStore)
	registerGetQuery(api, queryStore)
	registerUpdateQuery(api, queryStore)
	registerDeleteQuery(api, queryStore)
	registerBulkDeleteQueries(api, queryStore)
	registerQueryResults(api, queryStore)
	registerHostQueries(api, queryStore, hostStore)
	registerHostQueryResults(api, queryStore, hostStore)
}

func registerListQueries(api huma.API, queryStore *queries.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-queries",
		Method:      http.MethodGet,
		Path:        "/api/queries",
		Tags:        []string{queriesTag},
		Summary:     "List saved queries",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *queryListInput) (*queryListOutput, error) {
		items, count, err := queryStore.List(ctx, input.params())
		if err != nil {
			return nil, apihelpers.ResourceMutationError(queryResource, err)
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

func registerCreateQuery(api huma.API, queryStore *queries.QueryStore) {
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
		query, err := queryStore.Create(ctx, params)
		if err != nil {
			return nil, apihelpers.ResourceMutationError(queryResource, err)
		}
		return &queryOutput{Body: queryResponse(query)}, nil
	})
}

func registerGetQuery(api huma.API, queryStore *queries.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "get-query",
		Method:      http.MethodGet,
		Path:        queryIDPath,
		Tags:        []string{queriesTag},
		Summary:     "Get a saved query",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *queryGetInput) (*queryOutput, error) {
		id, err := apihelpers.ParseResourceID(input.ID, queryResource)
		if err != nil {
			return nil, err
		}
		query, err := queryStore.GetByID(ctx, id)
		if err != nil {
			return nil, apihelpers.ResourceMutationError(queryResource, err)
		}
		return &queryOutput{Body: queryResponse(query)}, nil
	})
}

func registerUpdateQuery(api huma.API, queryStore *queries.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "put-query",
		Method:      http.MethodPut,
		Path:        queryIDPath,
		Tags:        []string{queriesTag},
		Summary:     "Replace a saved query",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *queryPutInput) (*queryOutput, error) {
		id, err := apihelpers.ParseResourceID(input.ID, queryResource)
		if err != nil {
			return nil, err
		}
		params, err := input.Body.updateParams()
		if err != nil {
			return nil, err
		}
		query, err := queryStore.Update(ctx, id, params)
		if err != nil {
			return nil, apihelpers.ResourceMutationError(queryResource, err)
		}
		return &queryOutput{Body: queryResponse(query)}, nil
	})
}

func registerDeleteQuery(api huma.API, queryStore *queries.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-query",
		Method:      http.MethodDelete,
		Path:        queryIDPath,
		Tags:        []string{queriesTag},
		Summary:     "Delete a saved query",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *queryDeleteInput) (*struct{}, error) {
		id, err := apihelpers.ParseResourceID(input.ID, queryResource)
		if err != nil {
			return nil, err
		}
		if err := queryStore.Delete(ctx, id); err != nil {
			return nil, apihelpers.ResourceMutationError(queryResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteQueries(api huma.API, queryStore *queries.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-queries",
		Method:      http.MethodPost,
		Path:        "/api/queries/bulk-delete",
		Tags:        []string{queriesTag},
		Summary:     "Delete saved queries",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *queryBulkDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		ids, err := input.ids()
		if err != nil {
			return nil, err
		}
		if _, err := queryStore.DeleteMany(ctx, ids); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerQueryResults(api huma.API, queryStore *queries.QueryStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-query-results",
		Method:      http.MethodGet,
		Path:        "/api/queries/{id}/results",
		Tags:        []string{queriesTag},
		Summary:     "List latest report snapshots for a query",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *queryGetInput) (*queryResultsOutput, error) {
		id, err := apihelpers.ParseResourceID(input.ID, queryResource)
		if err != nil {
			return nil, err
		}
		rows, err := queryStore.Results(ctx, id)
		if err != nil {
			return nil, err
		}
		out := &queryResultsOutput{}
		out.Body.Items = queryResultResponses(rows)
		return out, nil
	})
}

func registerHostQueries(api huma.API, queryStore *queries.QueryStore, hostStore *hosts.HostStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-queries",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/queries",
		Tags:        []string{queriesTag, apihelpers.HostsTag},
		Summary:     "List reports for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostReportsOutput, error) {
		id, err := apihelpers.ParseHostID(input.ID)
		if err != nil {
			return nil, err
		}
		host, err := hostStore.GetByID(ctx, id)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		rows, err := queryStore.HostReports(ctx, host)
		if err != nil {
			return nil, err
		}
		out := &hostReportsOutput{}
		out.Body.Items = hostReportResponses(rows)
		return out, nil
	})
}

func registerHostQueryResults(api huma.API, queryStore *queries.QueryStore, hostStore *hosts.HostStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-query-results",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/queries/{query_id}",
		Tags:        []string{queriesTag, apihelpers.HostsTag},
		Summary:     "List report rows for one host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostQueryResultsInput) (*hostQueryResultsOutput, error) {
		hostID, err := apihelpers.ParseHostID(input.ID)
		if err != nil {
			return nil, err
		}
		queryID, err := apihelpers.ParseResourceID(input.QueryID, queryResource)
		if err != nil {
			return nil, err
		}
		host, err := hostStore.GetByID(ctx, hostID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		rows, lastFetched, err := queryStore.HostQueryResults(ctx, hostID, queryID)
		if err != nil {
			return nil, err
		}
		out := &hostQueryResultsOutput{}
		out.Body.QueryID = queryID
		out.Body.HostID = hostID
		out.Body.HostName = host.DisplayName
		out.Body.LastFetched = lastFetched
		out.Body.Items = queryResultResponses(rows)
		return out, nil
	})
}

func (input queryListInput) params() queries.QueryListParams {
	return queries.QueryListParams{
		ListParams: dbutil.CleanListParams(dbutil.ListParams{
			Q:              input.Q,
			Page:           input.Page,
			PerPage:        input.PerPage,
			OrderKey:       input.OrderKey,
			OrderDirection: input.OrderDirection,
		}),
		Platform: strings.TrimSpace(input.Platform),
	}
}

func (body queryMutationBody) createParams(userID *int64) (queries.QueryCreate, error) {
	s, err := body.LabelScope.model()
	if err != nil {
		return queries.QueryCreate{}, err
	}
	return queries.QueryCreate{
		Name:              body.Name,
		Description:       body.Description,
		Query:             body.Query,
		Platform:          body.Platform,
		MinOsqueryVersion: body.MinOsqueryVersion,
		ScheduleInterval:  body.ScheduleInterval,
		LoggingType:       queries.QueryLoggingSnapshot,
		LabelScope:        s,
		CreatedByUserID:   userID,
	}, nil
}

func (body queryMutationBody) updateParams() (queries.QueryUpdate, error) {
	s, err := body.LabelScope.model()
	if err != nil {
		return queries.QueryUpdate{}, err
	}
	return queries.QueryUpdate{
		Name:              body.Name,
		Description:       body.Description,
		Query:             body.Query,
		Platform:          body.Platform,
		MinOsqueryVersion: body.MinOsqueryVersion,
		ScheduleInterval:  body.ScheduleInterval,
		LoggingType:       queries.QueryLoggingSnapshot,
		LabelScope:        s,
	}, nil
}

func queryResponse(query *queries.Query) queryBody {
	return queryBody{
		ID:                query.ID,
		Name:              query.Name,
		Description:       query.Description,
		Query:             query.Query,
		Platform:          query.Platform,
		MinOsqueryVersion: query.MinOsqueryVersion,
		ScheduleInterval:  query.ScheduleInterval,
		LabelScope:        labelScopeResponse(query.LabelScope),
		CreatedByUserID:   query.CreatedByUserID,
		CreatedAt:         query.CreatedAt,
		UpdatedAt:         query.UpdatedAt,
	}
}

func queryResultResponses(rows []queries.QueryResult) []queryResultBody {
	out := make([]queryResultBody, 0, len(rows))
	for _, row := range rows {
		var lastFetched *time.Time
		if !row.LastFetched.IsZero() {
			lastFetched = &row.LastFetched
		}
		out = append(out, queryResultBody{
			QueryID:     row.QueryID,
			QueryName:   row.QueryName,
			HostID:      row.HostID,
			HostName:    row.HostName,
			Columns:     row.Columns,
			LastFetched: lastFetched,
		})
	}
	return out
}

func hostReportResponses(rows []queries.HostReport) []hostReportBody {
	out := make([]hostReportBody, 0, len(rows))
	for _, row := range rows {
		out = append(out, hostReportBody{
			ReportID:        row.ReportID,
			Name:            row.Name,
			Description:     row.Description,
			LastFetched:     row.LastFetched,
			FirstResult:     row.FirstResult,
			HostResultCount: row.HostResultCount,
		})
	}
	return out
}
