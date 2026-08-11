package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
)

const (
	checkResource = "check"
	checkIDPath   = "/api/osquery/checks/{id}"
)

type checkListInput struct {
	api.ListQueryInput
}

func (input checkListInput) params() checks.CheckListParams {
	return checks.CheckListParams{
		ListParams: input.Params(),
	}
}

type checkGetInput struct {
	ID int64 `path:"id"`
}

type checkResultsInput struct {
	api.ListQueryInput

	ID     int64                `path:"id"`
	Status []checks.CheckStatus `          query:"status,omitempty"`
}

func (input checkResultsInput) params() checks.CheckResultListParams {
	return checks.CheckResultListParams{
		ListParams: input.Params(),
		Statuses:   input.Status,
	}
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
	Body api.Page[checks.Check]
}

type checkOutput struct {
	Body checks.Check
}

type checkResultsOutput struct {
	Body api.Page[checks.CheckHostStatus]
}

func registerOsqueryChecks(humaAPI huma.API, checkStore *checks.Store, logger *slog.Logger) {
	registerListChecks(humaAPI, checkStore, logger)
	registerCreateCheck(humaAPI, checkStore, logger)
	registerGetCheck(humaAPI, checkStore, logger)
	registerUpdateCheck(humaAPI, checkStore, logger)
	registerDeleteCheck(humaAPI, checkStore, logger)
	registerBulkDeleteChecks(humaAPI, checkStore, logger)
	registerCheckResults(humaAPI, checkStore, logger)
}

func registerListChecks(humaAPI huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-osquery-checks",
		Method:      http.MethodGet,
		Path:        "/api/osquery/checks",
		Tags:        []string{api.TagOsqueryChecks},
		Summary:     "List checks",
	}, func(ctx context.Context, input *checkListInput) (*checkListOutput, error) {
		items, count, err := checkStore.List(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-osquery-checks", checkResource, err)
		}
		return &checkListOutput{Body: api.Page[checks.Check]{Items: items, Count: count}}, nil
	})
}

func registerCreateCheck(humaAPI huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-osquery-check",
		Method:        http.MethodPost,
		Path:          "/api/osquery/checks",
		Tags:          []string{api.TagOsqueryChecks},
		Summary:       "Create a check",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *checkCreateInput) (*checkOutput, error) {
		check, err := checkStore.Create(ctx, checks.CheckCreateMutation{
			CheckMutation:   input.Body,
			CreatedByUserID: ctxkeys.CurrentUserID(ctx),
		})
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-osquery-check", checkResource, err)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerGetCheck(humaAPI huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-osquery-check",
		Method:      http.MethodGet,
		Path:        checkIDPath,
		Tags:        []string{api.TagOsqueryChecks},
		Summary:     "Get a check",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *checkGetInput) (*checkOutput, error) {
		check, err := checkStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-osquery-check", checkResource, err, "id", input.ID)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerUpdateCheck(humaAPI huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-osquery-check",
		Method:      http.MethodPut,
		Path:        checkIDPath,
		Tags:        []string{api.TagOsqueryChecks},
		Summary:     "Update a check",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *checkPutInput) (*checkOutput, error) {
		check, err := checkStore.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(
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

func registerDeleteCheck(humaAPI huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-osquery-check",
		Method:      http.MethodDelete,
		Path:        checkIDPath,
		Tags:        []string{api.TagOsqueryChecks},
		Summary:     "Delete a check",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *checkDeleteInput) (*struct{}, error) {
		if err := checkStore.Delete(ctx, input.ID); err != nil {
			return nil, api.ResourceError(
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

func registerBulkDeleteChecks(humaAPI huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "bulk-delete-osquery-checks",
		Method:        http.MethodDelete,
		Path:          "/api/osquery/checks",
		Tags:          []string{api.TagOsqueryChecks},
		Summary:       "Delete checks",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *api.DeleteManyInput) (*struct{}, error) {
		if _, err := checkStore.DeleteMany(ctx, input.IDs); err != nil {
			return nil, api.HandlerError(ctx, logger, "bulk-delete-osquery-checks", err)
		}
		return &struct{}{}, nil
	})
}

func registerCheckResults(humaAPI huma.API, checkStore *checks.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-osquery-check-results",
		Method:      http.MethodGet,
		Path:        "/api/osquery/checks/{id}/results",
		Tags:        []string{api.TagOsqueryChecks},
		Summary:     "List check results",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *checkResultsInput) (*checkResultsOutput, error) {
		rows, count, err := checkStore.CheckResults(ctx, input.ID, input.params())
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"list-osquery-check-results",
				checkResource,
				err,
				"id",
				input.ID,
			)
		}
		return &checkResultsOutput{
			Body: api.Page[checks.CheckHostStatus]{Items: rows, Count: count},
		}, nil
	})
}
