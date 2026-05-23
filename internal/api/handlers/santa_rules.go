package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa"
)

const (
	santaRulesTag     = "Santa"
	santaRuleResource = "Santa rule"
	santaRuleIDPath   = "/api/santa/rules/{id}"
)

type santaRuleListInput struct {
	Q              string `query:"q,omitempty"`
	RuleType       string `query:"rule_type,omitempty"`
	Page           int    `query:"page,omitempty"`
	PerPage        int    `query:"per_page,omitempty"`
	OrderKey       string `query:"order_key,omitempty"`
	OrderDirection string `query:"order_direction,omitempty"`
}

type santaRuleGetInput struct {
	ID string `path:"id"`
}

type santaRuleCreateInput struct {
	Body santa.RuleCreate
}

type santaRulePatchInput struct {
	ID   string `path:"id"`
	Body santa.RuleUpdate
}

type santaRuleDeleteInput struct {
	ID string `path:"id"`
}

type santaRuleReorderIncludesInput struct {
	ID   string `path:"id"`
	Body santaRuleReorderIncludesBody
}

type santaRuleReorderIncludesBody struct {
	OrderedIncludeIDs []int64 `json:"ordered_include_ids"`
}

type santaRuleListOutput struct {
	Body paginatedBody[santa.Rule]
}

type santaRuleOutput struct {
	Body santa.Rule
}

func (input santaRuleListInput) params() santa.RuleListParams {
	return santa.RuleListParams{
		ListParams: dbutil.ListParams{
			Q:              input.Q,
			Page:           input.Page,
			PerPage:        input.PerPage,
			OrderKey:       input.OrderKey,
			OrderDirection: input.OrderDirection,
		},
		RuleType: santa.RuleType(input.RuleType),
	}
}

func RegisterSantaRules(api huma.API, store *santa.Store) {
	registerListSantaRules(api, store)
	registerCreateSantaRule(api, store)
	registerGetSantaRule(api, store)
	registerPatchSantaRule(api, store)
	registerDeleteSantaRule(api, store)
	registerReorderSantaRuleIncludes(api, store)
}

func registerListSantaRules(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-rules",
		Method:      http.MethodGet,
		Path:        "/api/santa/rules",
		Tags:        []string{santaRulesTag},
		Summary:     "List Santa rules",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaRuleListInput) (*santaRuleListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		rules, count, err := store.ListRules(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &santaRuleListOutput{Body: paginatedBody[santa.Rule]{Items: rules, Count: count}}, nil
	})
}

func registerCreateSantaRule(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-santa-rule",
		Method:        http.MethodPost,
		Path:          "/api/santa/rules",
		Tags:          []string{santaRulesTag},
		Summary:       "Create a Santa rule",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusConflict},
	}, func(ctx context.Context, input *santaRuleCreateInput) (*santaRuleOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		rule, err := store.CreateRule(ctx, input.Body)
		if err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerGetSantaRule(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-rule",
		Method:      http.MethodGet,
		Path:        santaRuleIDPath,
		Tags:        []string{santaRulesTag},
		Summary:     "Get a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleGetInput) (*santaRuleOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, santaRuleResource)
		if err != nil {
			return nil, err
		}
		rule, err := store.GetRuleByID(ctx, id)
		if err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerPatchSantaRule(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-santa-rule",
		Method:      http.MethodPatch,
		Path:        santaRuleIDPath,
		Tags:        []string{santaRulesTag},
		Summary:     "Update a Santa rule",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *santaRulePatchInput) (*santaRuleOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, santaRuleResource)
		if err != nil {
			return nil, err
		}
		rule, err := store.UpdateRule(ctx, id, input.Body)
		if err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerDeleteSantaRule(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-rule",
		Method:      http.MethodDelete,
		Path:        santaRuleIDPath,
		Tags:        []string{santaRulesTag},
		Summary:     "Delete a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, santaRuleResource)
		if err != nil {
			return nil, err
		}
		if err := store.DeleteRule(ctx, id); err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerReorderSantaRuleIncludes(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-santa-rule-includes",
		Method:      http.MethodPut,
		Path:        "/api/santa/rules/{id}/includes/order",
		Tags:        []string{santaRulesTag},
		Summary:     "Reorder Santa rule includes",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleReorderIncludesInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, santaRuleResource)
		if err != nil {
			return nil, err
		}
		if err := store.ReorderRuleIncludes(ctx, id, input.Body.OrderedIncludeIDs); err != nil {
			return nil, resourceMutationError(santaRuleResource, err)
		}
		return &struct{}{}, nil
	})
}
