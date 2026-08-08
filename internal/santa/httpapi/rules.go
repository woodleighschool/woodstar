package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
)

const (
	santaRuleResource = "Santa rule"
	santaRuleIDPath   = "/api/santa/rules/{id}"
)

type santaRuleListInput struct {
	api.ListQueryInput

	ConfigurationID []int64          `query:"configuration_id,omitempty"`
	RuleType        []rules.RuleType `query:"rule_type,omitempty"`
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

type santaRuleListOutput struct {
	Body api.Page[rules.Rule]
}

type santaRuleOutput struct {
	Body rules.Rule
}

func (input santaRuleListInput) params() rules.RuleListParams {
	return rules.RuleListParams{
		ListParams:       input.Params(),
		ConfigurationIDs: input.ConfigurationID,
		RuleTypes:        input.RuleType,
	}
}

func registerSantaRules(humaAPI huma.API, store *rules.Store, logger *slog.Logger) {
	registerListSantaRules(humaAPI, store, logger)
	registerCreateSantaRule(humaAPI, store, logger)
	registerGetSantaRule(humaAPI, store, logger)
	registerUpdateSantaRule(humaAPI, store, logger)
	registerDeleteSantaRule(humaAPI, store, logger)
	registerBulkDeleteSantaRules(humaAPI, store, logger)
}

func registerListSantaRules(humaAPI huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-santa-rules",
		Method:      http.MethodGet,
		Path:        "/api/santa/rules",
		Tags:        []string{api.TagSantaRules},
		Summary:     "List rules",
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *santaRuleListInput) (*santaRuleListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-santa-rules", santaRuleResource, err)
		}
		return &santaRuleListOutput{Body: api.Page[rules.Rule]{Items: rows, Count: count}}, nil
	})
}

func registerCreateSantaRule(humaAPI huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-santa-rule",
		Method:        http.MethodPost,
		Path:          "/api/santa/rules",
		Tags:          []string{api.TagSantaRules},
		Summary:       "Create a rule",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *santaRuleCreateInput) (*santaRuleOutput, error) {
		rule, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-santa-rule", santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerGetSantaRule(humaAPI huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-santa-rule",
		Method:      http.MethodGet,
		Path:        santaRuleIDPath,
		Tags:        []string{api.TagSantaRules},
		Summary:     "Get a rule",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleGetInput) (*santaRuleOutput, error) {
		rule, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-santa-rule", santaRuleResource, err, "id", input.ID)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerUpdateSantaRule(humaAPI huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-santa-rule",
		Method:      http.MethodPut,
		Path:        santaRuleIDPath,
		Tags:        []string{api.TagSantaRules},
		Summary:     "Update a rule",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *santaRuleUpdateInput) (*santaRuleOutput, error) {
		rule, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "update-santa-rule", santaRuleResource, err, "id", input.ID)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerDeleteSantaRule(humaAPI huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-santa-rule",
		Method:      http.MethodDelete,
		Path:        santaRuleIDPath,
		Tags:        []string{api.TagSantaRules},
		Summary:     "Delete a rule",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ID); err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-santa-rule", santaRuleResource, err, "id", input.ID)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteSantaRules(humaAPI huma.API, store *rules.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "bulk-delete-santa-rules",
		Method:        http.MethodDelete,
		Path:          "/api/santa/rules",
		Tags:          []string{api.TagSantaRules},
		Summary:       "Delete rules",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *api.DeleteManyInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.IDs); err != nil {
			return nil, api.ResourceError(ctx, logger, "bulk-delete-santa-rules", santaRuleResource, err)
		}
		return &struct{}{}, nil
	})
}
