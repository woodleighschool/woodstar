package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
)

const (
	checksTag     = "Checks"
	checkResource = "check"
	checkIDPath   = "/api/osquery/checks/{id}"
)

type checkListInput struct {
	ListQueryInput
}

type checkGetInput struct {
	CheckID int64 `path:"id"`
}

type checkResultsInput struct {
	CheckID  int64  `path:"id"`
	Response string `          query:"response,omitempty" enum:"pass,fail"`
}

type checkCreateInput struct {
	Body checks.CheckMutation
}

type checkPutInput struct {
	CheckID int64 `path:"id"`
	Body    checks.CheckMutation
}

type checkDeleteInput struct {
	CheckID int64 `path:"id"`
}

type checkBulkDeleteInput struct {
	Body BulkIDsBody
}

type checkListOutput struct {
	Body Page[checks.Check]
}

type checkOutput struct {
	Body checks.Check
}

type checkResultsOutput struct {
	Body []checks.CheckHostStatus
}

func registerOsqueryChecks(api huma.API, checkStore *checks.Store) {
	registerListChecks(api, checkStore)
	registerCreateCheck(api, checkStore)
	registerGetCheck(api, checkStore)
	registerUpdateCheck(api, checkStore)
	registerDeleteCheck(api, checkStore)
	registerBulkDeleteChecks(api, checkStore)
	registerCheckResults(api, checkStore)
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
			return nil, ResourceMutationError(checkResource, err)
		}
		return &checkListOutput{Body: Page[checks.Check]{Items: items, Count: int32(count)}}, nil
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
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *checkCreateInput) (*checkOutput, error) {
		check, err := checkStore.Create(ctx, checks.CheckCreateMutation{
			CheckMutation:   input.Body,
			CreatedByUserID: ctxkeys.CurrentUserID(ctx),
		})
		if err != nil {
			return nil, ResourceMutationError(checkResource, err)
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
		check, err := checkStore.GetByID(ctx, input.CheckID)
		if err != nil {
			return nil, ResourceMutationError(checkResource, err)
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
		check, err := checkStore.Update(ctx, input.CheckID, input.Body)
		if err != nil {
			return nil, ResourceMutationError(checkResource, err)
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
		if err := checkStore.Delete(ctx, input.CheckID); err != nil {
			return nil, ResourceMutationError(checkResource, err)
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
		if _, err := checkStore.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerCheckResults(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-check-results",
		Method:      http.MethodGet,
		Path:        "/api/osquery/checks/{id}/results",
		Tags:        []string{checksTag},
		Summary:     "List latest results for a check",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *checkResultsInput) (*checkResultsOutput, error) {
		var response *checks.CheckStatus
		if input.Response != "" {
			status := checks.CheckStatus(input.Response)
			response = &status
		}
		rows, err := checkStore.CheckResults(ctx, input.CheckID, response)
		if err != nil {
			return nil, err
		}
		return &checkResultsOutput{Body: rows}, nil
	})
}

func (input checkListInput) params() checks.CheckListParams {
	return checks.CheckListParams{
		ListParams: input.ListQueryInput.Params(),
	}
}
