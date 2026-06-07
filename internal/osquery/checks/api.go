package checks

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/adminctx"
	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
)

const (
	checksTag     = "Checks"
	checkResource = "check"
	checkIDPath   = "/api/osquery/checks/{check_id}"
)

type checkListInput struct {
	apitypes.ListQueryInput
}

type checkGetInput struct {
	CheckID int64 `path:"check_id"`
}

type checkCreateInput struct {
	Body CheckMutation
}

type checkPutInput struct {
	CheckID int64 `path:"check_id"`
	Body    CheckMutation
}

type checkDeleteInput struct {
	CheckID int64 `path:"check_id"`
}

type checkBulkDeleteInput struct {
	Body apitypes.BulkIDsBody
}

type checkListOutput struct {
	Body apitypes.Page[Check]
}

type checkOutput struct {
	Body Check
}

type checkHostsOutput struct {
	Body []CheckHostStatus
}

func RegisterAdminRoutes(api huma.API, checkStore *Store) {
	registerListChecks(api, checkStore)
	registerCreateCheck(api, checkStore)
	registerGetCheck(api, checkStore)
	registerUpdateCheck(api, checkStore)
	registerDeleteCheck(api, checkStore)
	registerBulkDeleteChecks(api, checkStore)
	registerCheckHosts(api, checkStore)
}

func registerListChecks(api huma.API, checkStore *Store) {
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
			return nil, apitypes.ResourceMutationError(checkResource, err)
		}
		return &checkListOutput{Body: apitypes.Page[Check]{Items: items, Count: count}}, nil
	})
}

func registerCreateCheck(api huma.API, checkStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-osquery-check",
		Method:        http.MethodPost,
		Path:          "/api/osquery/checks",
		Tags:          []string{checksTag},
		Summary:       "Create a check",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *checkCreateInput) (*checkOutput, error) {
		params := input.Body
		params.CreatedByUserID = adminctx.CurrentUserID(ctx)
		check, err := checkStore.Create(ctx, params)
		if err != nil {
			return nil, apitypes.ResourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerGetCheck(api huma.API, checkStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-osquery-check",
		Method:      http.MethodGet,
		Path:        checkIDPath,
		Tags:        []string{checksTag},
		Summary:     "Get a check",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *checkGetInput) (*checkOutput, error) {
		check, err := checkStore.GetByID(ctx, input.CheckID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerUpdateCheck(api huma.API, checkStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-osquery-check",
		Method:      http.MethodPut,
		Path:        checkIDPath,
		Tags:        []string{checksTag},
		Summary:     "Replace a check",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *checkPutInput) (*checkOutput, error) {
		params := input.Body
		check, err := checkStore.Update(ctx, input.CheckID, params)
		if err != nil {
			return nil, apitypes.ResourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerDeleteCheck(api huma.API, checkStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-osquery-check",
		Method:      http.MethodDelete,
		Path:        checkIDPath,
		Tags:        []string{checksTag},
		Summary:     "Delete a check",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *checkDeleteInput) (*struct{}, error) {
		if err := checkStore.Delete(ctx, input.CheckID); err != nil {
			return nil, apitypes.ResourceMutationError(checkResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteChecks(api huma.API, checkStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-osquery-checks",
		Method:      http.MethodPost,
		Path:        "/api/osquery/checks/bulk-delete",
		Tags:        []string{checksTag},
		Summary:     "Delete checks",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *checkBulkDeleteInput) (*struct{}, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		if _, err := checkStore.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerCheckHosts(api huma.API, checkStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-check-hosts",
		Method:      http.MethodGet,
		Path:        "/api/osquery/checks/{check_id}/hosts",
		Tags:        []string{checksTag},
		Summary:     "List check host status",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *checkGetInput) (*checkHostsOutput, error) {
		rows, err := checkStore.HostStatuses(ctx, input.CheckID)
		if err != nil {
			return nil, err
		}
		return &checkHostsOutput{Body: rows}, nil
	})
}

func (input checkListInput) params() CheckListParams {
	return CheckListParams{
		ListParams: input.ListQueryInput.Params(),
	}
}
