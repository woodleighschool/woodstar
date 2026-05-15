//nolint:dupl // Huma route registration is intentionally explicit per resource.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/agents/checks"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/scope"
)

const (
	checksTag     = "Checks"
	checkResource = "check"
	checkIDPath   = "/api/checks/{id}"
)

type checkMutationBody struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Query       string           `json:"query"`
	Platform    *string          `json:"platform,omitempty"`
	LabelScope  scope.LabelScope `json:"label_scope"`
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

type checkListOutput struct {
	Body struct {
		Items []checks.Check `json:"items"`
		Count int            `json:"count"`
	}
}

type checkOutput struct {
	Body checks.Check
}

type checkHostsOutput struct {
	Body struct {
		Items []checks.CheckHostStatus `json:"items"`
	}
}

// RegisterChecks registers check endpoints.
func RegisterChecks(api huma.API, checkStore *checks.Store, hostStore *hosts.Store) {
	registerListChecks(api, checkStore)
	registerCreateCheck(api, checkStore)
	registerGetCheck(api, checkStore)
	registerUpdateCheck(api, checkStore)
	registerDeleteCheck(api, checkStore)
	registerBulkDeleteChecks(api, checkStore)
	registerCheckHosts(api, checkStore)
	registerHostChecks(api, checkStore, hostStore)
}

func registerListChecks(api huma.API, checkStore *checks.Store) {
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
		out.Body.Items = items
		out.Body.Count = count
		return out, nil
	})
}

func registerCreateCheck(api huma.API, checkStore *checks.Store) {
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
		return &checkOutput{Body: *check}, nil
	})
}

func registerGetCheck(api huma.API, checkStore *checks.Store) {
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
		return &checkOutput{Body: *check}, nil
	})
}

func registerUpdateCheck(api huma.API, checkStore *checks.Store) {
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
		return &checkOutput{Body: *check}, nil
	})
}

func registerDeleteCheck(api huma.API, checkStore *checks.Store) {
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

func registerBulkDeleteChecks(api huma.API, checkStore *checks.Store) {
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
		ids, err := input.Body.ids("check IDs")
		if err != nil {
			return nil, err
		}
		if _, err := checkStore.DeleteMany(ctx, ids); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerCheckHosts(api huma.API, checkStore *checks.Store) {
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
		out.Body.Items = rows
		return out, nil
	})
}

func registerHostChecks(api huma.API, checkStore *checks.Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-checks",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/checks",
		Tags:        []string{checksTag, hostsTag},
		Summary:     "List checks for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*checkHostsOutput, error) {
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
		rows, err := checkStore.HostChecks(ctx, host)
		if err != nil {
			return nil, err
		}
		out := &checkHostsOutput{}
		out.Body.Items = rows
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
	s, err := normalizeLabelScope(body.LabelScope)
	if err != nil {
		return checks.CheckCreate{}, err
	}
	return checks.CheckCreate{
		Name:            body.Name,
		Description:     body.Description,
		Query:           body.Query,
		Platform:        body.Platform,
		LabelScope:      s,
		CreatedByUserID: userID,
	}, nil
}

func (body checkMutationBody) updateParams() (checks.CheckUpdate, error) {
	s, err := normalizeLabelScope(body.LabelScope)
	if err != nil {
		return checks.CheckUpdate{}, err
	}
	return checks.CheckUpdate{
		Name:        body.Name,
		Description: body.Description,
		Query:       body.Query,
		Platform:    body.Platform,
		LabelScope:  s,
	}, nil
}
