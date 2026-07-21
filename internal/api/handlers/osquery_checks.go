package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
)

const (
	checkResource = "check"
	checkIDPath   = "/api/osquery/checks/{id}"
)

type checkListInput struct {
	ListQueryInput
}

func (input checkListInput) params() checks.CheckListParams {
	return checks.CheckListParams{
		ListParams: input.ListQueryInput.params(),
	}
}

type checkGetInput struct {
	ID int64 `path:"id"`
}

type checkResultsInput struct {
	ID       int64  `path:"id"`
	Response string `          query:"response,omitempty" enum:"pass,fail"`
}

type checkCreateInput struct {
	Body checks.CheckMutation
}

type checkPutInput struct {
	ID   int64 `path:"id"`
	Body checks.CheckMutation
}

type checkDeleteInput struct {
	ID int64 `path:"id"`
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

func registerOsqueryChecks(api huma.API, checkStore *checks.Store, logger *slog.Logger) {
	registerListChecks(api, checkStore, logger)
	registerCreateCheck(api, checkStore, logger)
	registerGetCheck(api, checkStore, logger)
	registerUpdateCheck(api, checkStore, logger)
	registerDeleteCheck(api, checkStore, logger)
	registerBulkDeleteChecks(api, checkStore, logger)
	registerCheckResults(api, checkStore, logger)
}

func registerListChecks(api huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-checks",
		Method:      http.MethodGet,
		Path:        "/api/osquery/checks",
		Tags:        []string{osqueryChecksTag},
		Summary:     "List checks",
	}, func(ctx context.Context, input *checkListInput) (*checkListOutput, error) {
		items, count, err := checkStore.List(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-osquery-checks", checkResource, err)
		}
		return &checkListOutput{Body: Page[checks.Check]{Items: items, Count: count}}, nil
	})
}

func registerCreateCheck(api huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-osquery-check",
		Method:        http.MethodPost,
		Path:          "/api/osquery/checks",
		Tags:          []string{osqueryChecksTag},
		Summary:       "Create a check",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *checkCreateInput) (*checkOutput, error) {
		check, err := checkStore.Create(ctx, checks.CheckCreateMutation{
			CheckMutation:   input.Body,
			CreatedByUserID: ctxkeys.CurrentUserID(ctx),
		})
		if err != nil {
			return nil, resourceError(ctx, logger, "create-osquery-check", checkResource, err)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerGetCheck(api huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-osquery-check",
		Method:      http.MethodGet,
		Path:        checkIDPath,
		Tags:        []string{osqueryChecksTag},
		Summary:     "Get a check",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *checkGetInput) (*checkOutput, error) {
		check, err := checkStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceError(ctx, logger, "get-osquery-check", checkResource, err, "id", input.ID)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerUpdateCheck(api huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "update-osquery-check",
		Method:      http.MethodPut,
		Path:        checkIDPath,
		Tags:        []string{osqueryChecksTag},
		Summary:     "Update a check",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *checkPutInput) (*checkOutput, error) {
		check, err := checkStore.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"update-osquery-check",
				checkResource,
				err,
				"id",
				input.ID,
			)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerDeleteCheck(api huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-osquery-check",
		Method:      http.MethodDelete,
		Path:        checkIDPath,
		Tags:        []string{osqueryChecksTag},
		Summary:     "Delete a check",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *checkDeleteInput) (*struct{}, error) {
		if err := checkStore.Delete(ctx, input.ID); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"delete-osquery-check",
				checkResource,
				err,
				"id",
				input.ID,
			)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteChecks(api huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "bulk-delete-osquery-checks",
		Method:        http.MethodDelete,
		Path:          "/api/osquery/checks",
		Tags:          []string{osqueryChecksTag},
		Summary:       "Delete checks",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *deleteManyInput) (*struct{}, error) {
		if _, err := checkStore.DeleteMany(ctx, input.IDs); err != nil {
			return nil, handlerError(ctx, logger, "bulk-delete-osquery-checks", err)
		}
		return &struct{}{}, nil
	})
}

func registerCheckResults(api huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-check-results",
		Method:      http.MethodGet,
		Path:        "/api/osquery/checks/{id}/results",
		Tags:        []string{osqueryChecksTag},
		Summary:     "List check results",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *checkResultsInput) (*checkResultsOutput, error) {
		var response *checks.CheckStatus
		if input.Response != "" {
			status := checks.CheckStatus(input.Response)
			response = &status
		}
		rows, err := checkStore.CheckResults(ctx, input.ID, response)
		if err != nil {
			return nil, handlerError(ctx, logger, "list-osquery-check-results", err, "id", input.ID)
		}
		return &checkResultsOutput{Body: rows}, nil
	})
}
