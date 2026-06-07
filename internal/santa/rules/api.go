package rules

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
)

const (
	santaTag          = "Santa"
	santaRuleResource = "Santa rule"
	santaRuleIDPath   = "/api/santa/rules/{rule_id}"
)

type santaRuleListInput struct {
	apitypes.ListQueryInput
	RuleType RuleType `query:"rule_type,omitempty"`
}

type santaRuleTargetListInput struct {
	Q          string   `query:"q,omitempty"`
	TargetType RuleType `query:"target_type,omitempty"`
	Limit      int      `query:"limit,omitempty"       minimum:"1" maximum:"50"`
}

type santaRuleGetInput struct {
	RuleID int64 `path:"rule_id"`
}

type santaRuleCreateInput struct {
	Body RuleMutation
}

type santaRuleUpdateInput struct {
	RuleID int64 `path:"rule_id"`
	Body   RuleMutation
}

type santaRuleDeleteInput struct {
	RuleID int64 `path:"rule_id"`
}

type santaRuleBulkDeleteInput struct {
	Body apitypes.BulkIDsBody
}

type santaRuleListOutput struct {
	Body apitypes.Page[Rule]
}

type santaRuleOutput struct {
	Body Rule
}

type santaRuleTargetListOutput struct {
	Body []RuleTarget
}

func (input santaRuleListInput) params() RuleListParams {
	return RuleListParams{
		ListParams: input.ListQueryInput.Params(),
		RuleType:   input.RuleType,
	}
}

func (input santaRuleTargetListInput) params() RuleTargetListParams {
	return RuleTargetListParams(input)
}

func RegisterAdminRoutes(api huma.API, store *Store) {
	registerListSantaRules(api, store)
	registerListSantaRuleTargets(api, store)
	registerCreateSantaRule(api, store)
	registerGetSantaRule(api, store)
	registerUpdateSantaRule(api, store)
	registerDeleteSantaRule(api, store)
	registerBulkDeleteSantaRules(api, store)
}

func registerListSantaRules(api huma.API, store *Store) {
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
			return nil, apitypes.ResourceMutationError(santaRuleResource, err)
		}
		return &santaRuleListOutput{Body: apitypes.Page[Rule]{Items: rules, Count: count}}, nil
	})
}

func registerListSantaRuleTargets(api huma.API, store *Store) {
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
			return nil, apitypes.ResourceMutationError(santaRuleResource, err)
		}
		return &santaRuleTargetListOutput{Body: targets}, nil
	})
}

func registerCreateSantaRule(api huma.API, store *Store) {
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
			return nil, apitypes.ResourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerGetSantaRule(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-rule",
		Method:      http.MethodGet,
		Path:        santaRuleIDPath,
		Tags:        []string{santaTag},
		Summary:     "Get a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleGetInput) (*santaRuleOutput, error) {
		rule, err := store.GetRuleByID(ctx, input.RuleID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerUpdateSantaRule(api huma.API, store *Store) {
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
		rule, err := store.UpdateRule(ctx, input.RuleID, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerDeleteSantaRule(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-rule",
		Method:      http.MethodDelete,
		Path:        santaRuleIDPath,
		Tags:        []string{santaTag},
		Summary:     "Delete a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleDeleteInput) (*struct{}, error) {
		if err := store.DeleteRule(ctx, input.RuleID); err != nil {
			return nil, apitypes.ResourceMutationError(santaRuleResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteSantaRules(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-santa-rules",
		Method:      http.MethodPost,
		Path:        "/api/santa/rules/bulk-delete",
		Tags:        []string{santaTag},
		Summary:     "Delete Santa rules",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaRuleBulkDeleteInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, apitypes.ResourceMutationError(santaRuleResource, err)
		}
		return &struct{}{}, nil
	})
}
