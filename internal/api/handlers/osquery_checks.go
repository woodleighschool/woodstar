package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
)

const (
	checksTag     = "Checks"
	checkResource = "check"
	checkIDPath   = "/api/osquery/checks/{id}"
)

type checkListInput struct {
	ListQueryInput
	Platform string `query:"platform,omitempty"`
}

type checkGetInput struct {
	ID string `path:"id"`
}

type checkCreateInput struct {
	Body checks.CheckCreate
}

type checkPutInput struct {
	ID   string `path:"id"`
	Body checks.CheckUpdate
}

type checkDeleteInput struct {
	ID string `path:"id"`
}

type checkBulkDeleteInput struct {
	Body bulkIDsBody
}

type checkListOutput struct {
	Body paginatedBody[checks.Check]
}

type checkOutput struct {
	Body checks.Check
}

type checkHostsOutput struct {
	Body itemsBody[checks.CheckHostStatus]
}

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
		OperationID: "list-osquery-checks",
		Method:      http.MethodGet,
		Path:        "/api/osquery/checks",
		Tags:        []string{checksTag},
		Summary:     "List checks",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *checkListInput) (*checkListOutput, error) {
		items, count, err := checkStore.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &checkListOutput{Body: paginatedBody[checks.Check]{Items: items, Count: count}}, nil
	})
}

func registerCreateCheck(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-osquery-check",
		Method:        http.MethodPost,
		Path:          "/api/osquery/checks",
		Tags:          []string{checksTag},
		Summary:       "Create a check",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict},
	}, func(ctx context.Context, input *checkCreateInput) (*checkOutput, error) {
		params := input.Body
		params.CreatedByUserID = currentUserID(ctx)
		lscope, err := normalizeLabelScope(params.LabelScope)
		if err != nil {
			return nil, err
		}
		params.LabelScope = lscope
		check, err := checkStore.Create(ctx, params)
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerGetCheck(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-osquery-check",
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
		OperationID: "update-osquery-check",
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
		params := input.Body
		lscope, err := normalizeLabelScope(params.LabelScope)
		if err != nil {
			return nil, err
		}
		params.LabelScope = lscope
		check, err := checkStore.Update(ctx, id, params)
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerDeleteCheck(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-osquery-check",
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
		OperationID: "bulk-delete-osquery-checks",
		Method:      http.MethodPost,
		Path:        "/api/osquery/checks/bulk-delete",
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
		OperationID: "list-osquery-check-hosts",
		Method:      http.MethodGet,
		Path:        "/api/osquery/checks/{id}/hosts",
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
		return &checkHostsOutput{Body: itemsBody[checks.CheckHostStatus]{Items: rows}}, nil
	})
}

func registerHostChecks(api huma.API, checkStore *checks.Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-osquery-checks",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/checks",
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
		return &checkHostsOutput{Body: itemsBody[checks.CheckHostStatus]{Items: rows}}, nil
	})
}

func (input checkListInput) params() checks.CheckListParams {
	return checks.CheckListParams{
		ListParams: input.ListQueryInput.params(),
		Platform:   input.Platform,
	}
}
