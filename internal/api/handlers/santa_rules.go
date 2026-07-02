package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/santa/rules"
)

const (
	santaRuleResource = "Santa rule"
	santaRuleIDPath   = "/api/santa/rules/{id}"
)

type santaRuleListInput struct {
	ListQueryInput

	RuleType rules.RuleType `query:"rule_type,omitempty"`
}

type santaRuleGetInput struct {
	ID int64 `path:"id"`
}

type santaRuleCreateInput struct {
	Body rules.RuleMutation
}

type santaRuleUpdateInput struct {
	ID   int64 `path:"id"`
	Body rules.RuleMutation
}

type santaRuleDeleteInput struct {
	ID int64 `path:"id"`
}

type santaRuleBulkDeleteInput struct {
	Body BulkIDsBody
}

type santaRuleListOutput struct {
	Body Page[rules.Rule]
}

type santaRuleOutput struct {
	Body rules.Rule
}

func (input santaRuleListInput) params() rules.RuleListParams {
	return rules.RuleListParams{
		ListParams: input.ListQueryInput.params(),
		RuleType:   input.RuleType,
	}
}

func registerSantaRules(api huma.API, store *rules.Store, logger *slog.Logger) {
	registerListSantaRules(api, store, logger)
	registerCreateSantaRule(api, store, logger)
	registerGetSantaRule(api, store, logger)
	registerUpdateSantaRule(api, store, logger)
	registerDeleteSantaRule(api, store, logger)
	registerBulkDeleteSantaRules(api, store, logger)
}

func registerListSantaRules(api huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-rules",
		Method:      http.MethodGet,
		Path:        "/api/santa/rules",
		Tags:        []string{santaTag},
		Summary:     "List Santa rules",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized},
	}, func(ctx context.Context, input *santaRuleListInput) (*santaRuleListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-santa-rules", santaRuleResource, err)
		}
		return &santaRuleListOutput{Body: Page[rules.Rule]{Items: rows, Count: count}}, nil
	})
}

func registerCreateSantaRule(api huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-santa-rule",
		Method:        http.MethodPost,
		Path:          "/api/santa/rules",
		Tags:          []string{santaTag},
		Summary:       "Create a Santa rule",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *santaRuleCreateInput) (*santaRuleOutput, error) {
		rule, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, resourceError(ctx, logger, "create-santa-rule", santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerGetSantaRule(api huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-rule",
		Method:      http.MethodGet,
		Path:        santaRuleIDPath,
		Tags:        []string{santaTag},
		Summary:     "Get a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleGetInput) (*santaRuleOutput, error) {
		rule, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceError(ctx, logger, "get-santa-rule", santaRuleResource, err, "id", input.ID)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerUpdateSantaRule(api huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "update-santa-rule",
		Method:      http.MethodPut,
		Path:        santaRuleIDPath,
		Tags:        []string{santaTag},
		Summary:     "Update a Santa rule",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *santaRuleUpdateInput) (*santaRuleOutput, error) {
		rule, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceError(ctx, logger, "update-santa-rule", santaRuleResource, err, "id", input.ID)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerDeleteSantaRule(api huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-rule",
		Method:      http.MethodDelete,
		Path:        santaRuleIDPath,
		Tags:        []string{santaTag},
		Summary:     "Delete a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ID); err != nil {
			return nil, resourceError(ctx, logger, "delete-santa-rule", santaRuleResource, err, "id", input.ID)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteSantaRules(api huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-santa-rules",
		Method:      http.MethodPost,
		Path:        "/api/santa/rules/bulk-delete",
		Tags:        []string{santaTag},
		Summary:     "Delete Santa rules",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaRuleBulkDeleteInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, resourceError(ctx, logger, "bulk-delete-santa-rules", santaRuleResource, err)
		}
		return &struct{}{}, nil
	})
}
