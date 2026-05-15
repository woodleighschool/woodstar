//nolint:dupl // Huma route registration is intentionally explicit per resource.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/agents/queries"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/scope"
)

const (
	queriesTag    = "Queries"
	queryResource = "query"
	queryIDPath   = "/api/queries/{id}"
)

type queryMutationBody struct {
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	Query             string           `json:"query"`
	Platform          *string          `json:"platform,omitempty"`
	MinOsqueryVersion *string          `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int              `json:"schedule_interval,omitempty"`
	LabelScope        scope.LabelScope `json:"label_scope"`
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

type queryListOutput struct {
	Body struct {
		Items []queries.Query `json:"items"`
		Count int             `json:"count"`
	}
}

type queryOutput struct {
	Body queries.Query
}

type queryResultsOutput struct {
	Body struct {
		Items []queries.QueryResult `json:"items"`
	}
}

type hostReportsOutput struct {
	Body struct {
		Items []queries.HostReport `json:"items"`
	}
}

type hostQueryResultsOutput struct {
	Body struct {
		QueryID     int64                 `json:"query_id"`
		HostID      int64                 `json:"host_id"`
		HostName    string                `json:"host_name"`
		LastFetched *time.Time            `json:"last_fetched,omitempty"`
		Items       []queries.QueryResult `json:"items"`
	}
}

type hostQueryResultsInput struct {
	ID      string `path:"id"`
	QueryID string `path:"query_id"`
}

// RegisterQueries registers saved-query and report endpoints.
func RegisterQueries(api huma.API, queryStore *queries.Store, hostStore *hosts.Store) {
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

func registerListQueries(api huma.API, queryStore *queries.Store) {
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
			return nil, resourceMutationError(queryResource, err)
		}
		out := &queryListOutput{}
		out.Body.Items = items
		out.Body.Count = count
		return out, nil
	})
}

func registerCreateQuery(api huma.API, queryStore *queries.Store) {
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
			return nil, resourceMutationError(queryResource, err)
		}
		return &queryOutput{Body: *query}, nil
	})
}

func registerGetQuery(api huma.API, queryStore *queries.Store) {
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
		query, err := queryStore.GetByID(ctx, id)
		if err != nil {
			return nil, resourceMutationError(queryResource, err)
		}
		return &queryOutput{Body: *query}, nil
	})
}

func registerUpdateQuery(api huma.API, queryStore *queries.Store) {
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
		query, err := queryStore.Update(ctx, id, params)
		if err != nil {
			return nil, resourceMutationError(queryResource, err)
		}
		return &queryOutput{Body: *query}, nil
	})
}

func registerDeleteQuery(api huma.API, queryStore *queries.Store) {
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
		if err := queryStore.Delete(ctx, id); err != nil {
			return nil, resourceMutationError(queryResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteQueries(api huma.API, queryStore *queries.Store) {
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
		ids, err := input.Body.ids("query IDs")
		if err != nil {
			return nil, err
		}
		if _, err := queryStore.DeleteMany(ctx, ids); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerQueryResults(api huma.API, queryStore *queries.Store) {
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
		rows, err := queryStore.Results(ctx, id)
		if err != nil {
			return nil, err
		}
		out := &queryResultsOutput{}
		out.Body.Items = rows
		return out, nil
	})
}

func registerHostQueries(api huma.API, queryStore *queries.Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-queries",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/queries",
		Tags:        []string{queriesTag, hostsTag},
		Summary:     "List reports for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostReportsOutput, error) {
		id, err := parseResourceID(input.ID, hostResource)
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
		out.Body.Items = rows
		return out, nil
	})
}

func registerHostQueryResults(api huma.API, queryStore *queries.Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-query-results",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/queries/{query_id}",
		Tags:        []string{queriesTag, hostsTag},
		Summary:     "List report rows for one host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostQueryResultsInput) (*hostQueryResultsOutput, error) {
		hostID, err := parseResourceID(input.ID, hostResource)
		if err != nil {
			return nil, err
		}
		queryID, err := parseResourceID(input.QueryID, queryResource)
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
		out.Body.Items = rows
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
	s, err := normalizeLabelScope(body.LabelScope)
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
		LabelScope:        s,
		CreatedByUserID:   userID,
	}, nil
}

func (body queryMutationBody) updateParams() (queries.QueryUpdate, error) {
	s, err := normalizeLabelScope(body.LabelScope)
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
		LabelScope:        s,
	}, nil
}
