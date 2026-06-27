package handlers

import (
	"context"
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

type santaRuleReferenceListInput struct {
	Q        string         `query:"q,omitempty"`
	RuleType rules.RuleType `query:"rule_type,omitempty"`
	Limit    int32          `query:"limit,omitempty"     minimum:"1" maximum:"50"`
}

type santaRuleGetInput struct {
	RuleID int64 `path:"id"`
}

type santaRuleCreateInput struct {
	Body rules.RuleMutation
}

type santaRuleUpdateInput struct {
	RuleID int64 `path:"id"`
	Body   rules.RuleMutation
}

type santaRuleDeleteInput struct {
	RuleID int64 `path:"id"`
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

type santaRuleReferenceListOutput struct {
	Body []rules.RuleReferenceCandidate
}

func (input santaRuleListInput) params() rules.RuleListParams {
	return rules.RuleListParams{
		ListParams: input.ListQueryInput.Params(),
		RuleType:   input.RuleType,
	}
}

func (input santaRuleReferenceListInput) params() rules.RuleReferenceListParams {
	return rules.RuleReferenceListParams(input)
}

func registerSantaRules(api huma.API, store *rules.Store) {
	registerListSantaRules(api, store)
	registerListSantaRuleReferences(api, store)
	registerCreateSantaRule(api, store)
	registerGetSantaRule(api, store)
	registerUpdateSantaRule(api, store)
	registerDeleteSantaRule(api, store)
	registerBulkDeleteSantaRules(api, store)
}

func registerListSantaRules(api huma.API, store *rules.Store) {
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
			return nil, ResourceMutationError(santaRuleResource, err)
		}
		return &santaRuleListOutput{Body: Page[rules.Rule]{Items: rows, Count: int32(count)}}, nil
	})
}

func registerListSantaRuleReferences(api huma.API, store *rules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-rule-references",
		Method:      http.MethodGet,
		Path:        "/api/santa/rule-references",
		Tags:        []string{santaTag},
		Summary:     "List Santa rule references",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized},
	}, func(ctx context.Context, input *santaRuleReferenceListInput) (*santaRuleReferenceListOutput, error) {
		candidates, err := store.ListRuleReferences(ctx, input.params())
		if err != nil {
			return nil, ResourceMutationError(santaRuleResource, err)
		}
		return &santaRuleReferenceListOutput{Body: candidates}, nil
	})
}

func registerCreateSantaRule(api huma.API, store *rules.Store) {
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
			return nil, ResourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerGetSantaRule(api huma.API, store *rules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-rule",
		Method:      http.MethodGet,
		Path:        santaRuleIDPath,
		Tags:        []string{santaTag},
		Summary:     "Get a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleGetInput) (*santaRuleOutput, error) {
		rule, err := store.GetByID(ctx, input.RuleID)
		if err != nil {
			return nil, ResourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerUpdateSantaRule(api huma.API, store *rules.Store) {
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
		rule, err := store.Update(ctx, input.RuleID, input.Body)
		if err != nil {
			return nil, ResourceMutationError(santaRuleResource, err)
		}
		return &santaRuleOutput{Body: *rule}, nil
	})
}

func registerDeleteSantaRule(api huma.API, store *rules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-rule",
		Method:      http.MethodDelete,
		Path:        santaRuleIDPath,
		Tags:        []string{santaTag},
		Summary:     "Delete a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.RuleID); err != nil {
			return nil, ResourceMutationError(santaRuleResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteSantaRules(api huma.API, store *rules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-santa-rules",
		Method:      http.MethodPost,
		Path:        "/api/santa/rules/bulk-delete",
		Tags:        []string{santaTag},
		Summary:     "Delete Santa rules",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaRuleBulkDeleteInput) (*struct{}, error) {
		if _, err := store.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, ResourceMutationError(santaRuleResource, err)
		}
		return &struct{}{}, nil
	})
}
