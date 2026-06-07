package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
)

const (
	santaRuleResource = "Santa rule"
	santaRuleIDPath   = "/api/santa/rules/{id}"
)

type santaRuleListInput struct {
	ListQueryInput
	RuleType santarules.RuleType `query:"rule_type,omitempty"`
}

type santaRuleTargetListInput struct {
	Q          string              `query:"q,omitempty"`
	TargetType santarules.RuleType `query:"target_type,omitempty"`
	Limit      int                 `query:"limit,omitempty"       minimum:"1" maximum:"50"`
}

type santaRuleGetInput struct {
	ID int64 `path:"id"`
}

type santaRuleCreateInput struct {
	Body santarules.RuleMutation
}

type santaRulePatchInput struct {
	ID   int64 `path:"id"`
	Body santarules.RuleMutation
}

type santaRuleDeleteInput struct {
	ID int64 `path:"id"`
}

type santaRuleBulkDeleteInput struct {
	Body bulkIDsBody
}

type santaRuleListOutput struct {
	Body Page[santarules.Rule]
}

type santaRuleOutput struct {
	Body santarules.Rule
}

type santaRuleTargetListOutput struct {
	Body []santarules.RuleTarget
}

func (input santaRuleListInput) params() santarules.RuleListParams {
	return santarules.RuleListParams{
		ListParams: input.ListQueryInput.params(),
		RuleType:   input.RuleType,
	}
}

func (input santaRuleTargetListInput) params() santarules.RuleTargetListParams {
	return santarules.RuleTargetListParams{
		Q:          input.Q,
		TargetType: input.TargetType,
		Limit:      input.Limit,
	}
}

func RegisterSantaRules(api huma.API, store *santarules.Store) {
	registerListSantaRules(api, store)
	registerListSantaRuleTargets(api, store)
	registerCreateSantaRule(api, store)
	registerGetSantaRule(api, store)
	registerPatchSantaRule(api, store)
	registerDeleteSantaRule(api, store)
	registerBulkDeleteSantaRules(api, store)
}

func registerListSantaRules(api huma.API, store *santarules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-rules",
		Method:      http.MethodGet,
		Path:        "/api/santa/rules",
		Tags:        []string{santaTag},
		Summary:     "List Santa rules",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaRuleListInput) (*santaRuleListOutput, error) {
		rules, count, err := store.ListRules(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &santaRuleListOutput{Body: Page[santarules.Rule]{Items: rules, Count: count}}, nil
	})
}

func registerListSantaRuleTargets(api huma.API, store *santarules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-rule-targets",
		Method:      http.MethodGet,
		Path:        "/api/santa/rule-targets",
		Tags:        []string{santaTag},
		Summary:     "List Santa rule targets",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaRuleTargetListInput) (*santaRuleTargetListOutput, error) {
		targets, err := store.ListRuleTargets(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &santaRuleTargetListOutput{Body: targets}, nil
	})
}

func registerCreateSantaRule(api huma.API, store *santarules.Store) {
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
		rule, err := store.CreateRule(ctx, input.Body)
		if err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerGetSantaRule(api huma.API, store *santarules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-rule",
		Method:      http.MethodGet,
		Path:        santaRuleIDPath,
		Tags:        []string{santaTag},
		Summary:     "Get a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleGetInput) (*santaRuleOutput, error) {
		rule, err := store.GetRuleByID(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerPatchSantaRule(api huma.API, store *santarules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-santa-rule",
		Method:      http.MethodPatch,
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
	}, func(ctx context.Context, input *santaRulePatchInput) (*santaRuleOutput, error) {
		rule, err := store.UpdateRule(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerDeleteSantaRule(api huma.API, store *santarules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-rule",
		Method:      http.MethodDelete,
		Path:        santaRuleIDPath,
		Tags:        []string{santaTag},
		Summary:     "Delete a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleDeleteInput) (*struct{}, error) {
		if err := store.DeleteRule(ctx, input.ID); err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteSantaRules(api huma.API, store *santarules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-santa-rules",
		Method:      http.MethodPost,
		Path:        "/api/santa/rules/bulk-delete",
		Tags:        []string{santaTag},
		Summary:     "Delete Santa rules",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaRuleBulkDeleteInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &struct{}{}, nil
	})
}
