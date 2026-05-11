//nolint:dupl // Huma route registration is intentionally explicit per resource.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/agents/checks"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

const (
	checksTag     = "Checks"
	checkResource = "check"
	checkIDPath   = "/api/checks/{id}"
)

type checkBody struct {
	ID                int64          `json:"id"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	Query             string         `json:"query"`
	Platform          *string        `json:"platform,omitempty"`
	MinOsqueryVersion *string        `json:"min_osquery_version,omitempty"`
	LabelScope        labelScopeBody `json:"label_scope,omitzero"`
	CreatedByUserID   *int64         `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type checkMutationBody struct {
	Name              string         `json:"name"`
	Description       string         `json:"description,omitempty"`
	Query             string         `json:"query"`
	Platform          *string        `json:"platform,omitempty"`
	MinOsqueryVersion *string        `json:"min_osquery_version,omitempty"`
	LabelScope        labelScopeBody `json:"label_scope"`
}

type checkListInput struct {
	Q              string `query:"q,omitempty"`
	Platform       string `query:"platform,omitempty"`
	Page           int    `query:"page,omitempty"`
	PerPage        int    `query:"per_page,omitempty"`
	OrderKey       string `query:"order_key,omitempty"`
	OrderDirection string `query:"order_direction,omitempty"`
}

type checkGetInput struct {
	ID string `path:"id"`
}

type checkCreateInput struct {
	Body checkMutationBody
}

type checkPutInput struct {
	ID   string `path:"id"`
	Body checkMutationBody
}

type checkDeleteInput struct {
	ID string `path:"id"`
}

type checkBulkDeleteInput struct {
	Body bulkIDsBody
}

func (i checkBulkDeleteInput) ids() ([]int64, error) {
	return cleanBulkIDs(i.Body.IDs, "check IDs")
}

type checkListOutput struct {
	Body struct {
		Items []checkBody `json:"items"`
		Count int         `json:"count"`
	}
}

type checkOutput struct {
	Body checkBody
}

type checkHostsOutput struct {
	Body struct {
		Items []checkHostBody `json:"items"`
	}
}

type checkHostBody struct {
	CheckID         int64      `json:"check_id"`
	CheckName       string     `json:"check_name"`
	HostID          int64      `json:"host_id"`
	HostName        string     `json:"host_name"`
	Passes          *bool      `json:"passes,omitempty"`
	FirstFailedAt   *time.Time `json:"first_failed_at,omitempty"`
	LastEvaluatedAt *time.Time `json:"last_evaluated_at,omitempty"`
}

// RegisterChecks registers check endpoints.
func RegisterChecks(api huma.API, checkStore *checks.CheckStore, hostStore *hosts.HostStore) {
	registerListChecks(api, checkStore)
	registerCreateCheck(api, checkStore)
	registerGetCheck(api, checkStore)
	registerUpdateCheck(api, checkStore)
	registerDeleteCheck(api, checkStore)
	registerBulkDeleteChecks(api, checkStore)
	registerCheckHosts(api, checkStore)
	registerHostChecks(api, checkStore, hostStore)
}

func registerListChecks(api huma.API, checkStore *checks.CheckStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-checks",
		Method:      http.MethodGet,
		Path:        "/api/checks",
		Tags:        []string{checksTag},
		Summary:     "List checks",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *checkListInput) (*checkListOutput, error) {
		items, count, err := checkStore.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		out := &checkListOutput{}
		out.Body.Items = make([]checkBody, 0, len(items))
		out.Body.Count = count
		for i := range items {
			out.Body.Items = append(out.Body.Items, checkResponse(&items[i]))
		}
		return out, nil
	})
}

func registerCreateCheck(api huma.API, checkStore *checks.CheckStore) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-check",
		Method:        http.MethodPost,
		Path:          "/api/checks",
		Tags:          []string{checksTag},
		Summary:       "Create a check",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict},
	}, func(ctx context.Context, input *checkCreateInput) (*checkOutput, error) {
		params, err := input.Body.createParams(currentUserID(ctx))
		if err != nil {
			return nil, err
		}
		check, err := checkStore.Create(ctx, params)
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: checkResponse(check)}, nil
	})
}

func registerGetCheck(api huma.API, checkStore *checks.CheckStore) {
	huma.Register(api, huma.Operation{
		OperationID: "get-check",
		Method:      http.MethodGet,
		Path:        checkIDPath,
		Tags:        []string{checksTag},
		Summary:     "Get a check",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *checkGetInput) (*checkOutput, error) {
		id, err := parseResourceID(input.ID, checkResource)
		if err != nil {
			return nil, err
		}
		check, err := checkStore.GetByID(ctx, id)
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: checkResponse(check)}, nil
	})
}

func registerUpdateCheck(api huma.API, checkStore *checks.CheckStore) {
	huma.Register(api, huma.Operation{
		OperationID: "put-check",
		Method:      http.MethodPut,
		Path:        checkIDPath,
		Tags:        []string{checksTag},
		Summary:     "Replace a check",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *checkPutInput) (*checkOutput, error) {
		id, err := parseResourceID(input.ID, checkResource)
		if err != nil {
			return nil, err
		}
		params, err := input.Body.updateParams()
		if err != nil {
			return nil, err
		}
		check, err := checkStore.Update(ctx, id, params)
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: checkResponse(check)}, nil
	})
}

func registerDeleteCheck(api huma.API, checkStore *checks.CheckStore) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-check",
		Method:      http.MethodDelete,
		Path:        checkIDPath,
		Tags:        []string{checksTag},
		Summary:     "Delete a check",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *checkDeleteInput) (*struct{}, error) {
		id, err := parseResourceID(input.ID, checkResource)
		if err != nil {
			return nil, err
		}
		if err := checkStore.Delete(ctx, id); err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteChecks(api huma.API, checkStore *checks.CheckStore) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-checks",
		Method:      http.MethodPost,
		Path:        "/api/checks/bulk-delete",
		Tags:        []string{checksTag},
		Summary:     "Delete checks",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *checkBulkDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		ids, err := input.ids()
		if err != nil {
			return nil, err
		}
		if _, err := checkStore.DeleteMany(ctx, ids); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerCheckHosts(api huma.API, checkStore *checks.CheckStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-check-hosts",
		Method:      http.MethodGet,
		Path:        "/api/checks/{id}/hosts",
		Tags:        []string{checksTag},
		Summary:     "List check host status",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *checkGetInput) (*checkHostsOutput, error) {
		id, err := parseResourceID(input.ID, checkResource)
		if err != nil {
			return nil, err
		}
		rows, err := checkStore.HostStatuses(ctx, id)
		if err != nil {
			return nil, err
		}
		out := &checkHostsOutput{}
		out.Body.Items = checkHostResponses(rows)
		return out, nil
	})
}

func registerHostChecks(api huma.API, checkStore *checks.CheckStore, hostStore *hosts.HostStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-checks",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/checks",
		Tags:        []string{checksTag, hostsTag},
		Summary:     "List checks for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*checkHostsOutput, error) {
		id, err := parseHostID(input.ID)
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
		rows, err := checkStore.HostChecks(ctx, *host)
		if err != nil {
			return nil, err
		}
		out := &checkHostsOutput{}
		out.Body.Items = checkHostResponses(rows)
		return out, nil
	})
}

func (input checkListInput) params() checks.CheckListParams {
	return checks.CheckListParams{
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

func (body checkMutationBody) createParams(userID *int64) (checks.CheckCreate, error) {
	scope, err := body.LabelScope.model()
	if err != nil {
		return checks.CheckCreate{}, err
	}
	return checks.CheckCreate{
		Name:              body.Name,
		Description:       body.Description,
		Query:             body.Query,
		Platform:          body.Platform,
		MinOsqueryVersion: body.MinOsqueryVersion,
		LabelScope:        scope,
		CreatedByUserID:   userID,
	}, nil
}

func (body checkMutationBody) updateParams() (checks.CheckUpdate, error) {
	scope, err := body.LabelScope.model()
	if err != nil {
		return checks.CheckUpdate{}, err
	}
	return checks.CheckUpdate{
		Name:              body.Name,
		Description:       body.Description,
		Query:             body.Query,
		Platform:          body.Platform,
		MinOsqueryVersion: body.MinOsqueryVersion,
		LabelScope:        scope,
	}, nil
}

func checkResponse(check *checks.Check) checkBody {
	return checkBody{
		ID:                check.ID,
		Name:              check.Name,
		Description:       check.Description,
		Query:             check.Query,
		Platform:          check.Platform,
		MinOsqueryVersion: check.MinOsqueryVersion,
		LabelScope:        labelScopeResponse(check.LabelScope),
		CreatedByUserID:   check.CreatedByUserID,
		CreatedAt:         check.CreatedAt,
		UpdatedAt:         check.UpdatedAt,
	}
}

func checkHostResponses(rows []checks.CheckHostStatus) []checkHostBody {
	out := make([]checkHostBody, 0, len(rows))
	for _, row := range rows {
		out = append(out, checkHostBody{
			CheckID:         row.CheckID,
			CheckName:       row.CheckName,
			HostID:          row.HostID,
			HostName:        row.HostName,
			Passes:          row.Passes,
			FirstFailedAt:   row.FirstFailedAt,
			LastEvaluatedAt: row.LastEvaluatedAt,
		})
	}
	return out
}
