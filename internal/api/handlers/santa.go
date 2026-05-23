package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	santasync "github.com/woodleighschool/woodstar/internal/santa/sync"
)

const (
	santaTag                   = "Santa"
	santaConfigurationResource = "Santa configuration"
	santaConfigurationIDPath   = "/api/santa/configurations/{id}"
	santaRuleResource          = "Santa rule"
	santaRuleIDPath            = "/api/santa/rules/{id}"
)

// Host detail Santa enrichment.

type santaHostDetailEnricher struct {
	store *santa.Store
}

// SantaHostDetailEnricher returns a host detail enricher backed by Santa state.
func SantaHostDetailEnricher(store *santa.Store) HostDetailEnricher {
	if store == nil {
		return nil
	}
	return santaHostDetailEnricher{store: store}
}

func (e santaHostDetailEnricher) EnrichHostDetail(ctx context.Context, hostID int64, body *hostDetailBody) error {
	detail, err := e.store.LoadHostState(ctx, hostID)
	if err != nil {
		return err
	}
	body.Santa = detail
	return nil
}

// Santa configurations.

type santaConfigurationListInput struct {
	Q              string `query:"q,omitempty"`
	Page           int    `query:"page,omitempty"`
	PerPage        int    `query:"per_page,omitempty"`
	OrderKey       string `query:"order_key,omitempty"`
	OrderDirection string `query:"order_direction,omitempty"`
}

type santaConfigurationGetInput struct {
	ID string `path:"id"`
}

type santaConfigurationCreateInput struct {
	Body configurations.ConfigurationMutation
}

type santaConfigurationPatchInput struct {
	ID   string `path:"id"`
	Body configurations.ConfigurationMutation
}

type santaConfigurationDeleteInput struct {
	ID string `path:"id"`
}

type santaConfigurationReorderInput struct {
	Body santaConfigurationReorderBody
}

type santaConfigurationReorderBody struct {
	OrderedIDs []int64 `json:"ordered_ids"`
}

type santaConfigurationListOutput struct {
	Body paginatedBody[configurations.Configuration]
}

type santaConfigurationOutput struct {
	Body configurations.Configuration
}

type santaConfigurationConflictError struct {
	LabelID           int64  `json:"label_id"`
	ConfigurationID   int64  `json:"configuration_id"`
	ConfigurationName string `json:"configuration_name"`
}

func (e santaConfigurationConflictError) Error() string {
	return "configuration label already belongs to another configuration"
}

func (e santaConfigurationConflictError) GetStatus() int {
	return http.StatusConflict
}

func (input santaConfigurationListInput) params() configurations.ConfigurationListParams {
	return configurations.ConfigurationListParams{
		ListParams: dbutil.ListParams{
			Q:              input.Q,
			Page:           input.Page,
			PerPage:        input.PerPage,
			OrderKey:       input.OrderKey,
			OrderDirection: input.OrderDirection,
		},
	}
}

func RegisterSantaConfigurations(api huma.API, store *configurations.Store) {
	registerListSantaConfigurations(api, store)
	registerCreateSantaConfiguration(api, store)
	registerGetSantaConfiguration(api, store)
	registerPatchSantaConfiguration(api, store)
	registerDeleteSantaConfiguration(api, store)
	registerReorderSantaConfigurations(api, store)
}

func registerListSantaConfigurations(api huma.API, store *configurations.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-configurations",
		Method:      http.MethodGet,
		Path:        "/api/santa/configurations",
		Tags:        []string{santaTag},
		Summary:     "List Santa configurations",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaConfigurationListInput) (*santaConfigurationListOutput, error) {
		rows, count, err := store.ListConfigurations(ctx, input.params())
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationListOutput{
			Body: paginatedBody[configurations.Configuration]{Items: rows, Count: count},
		}, nil
	})
}

func registerCreateSantaConfiguration(api huma.API, store *configurations.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-santa-configuration",
		Method:        http.MethodPost,
		Path:          "/api/santa/configurations",
		Tags:          []string{santaTag},
		Summary:       "Create a Santa configuration",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusConflict},
	}, func(ctx context.Context, input *santaConfigurationCreateInput) (*santaConfigurationOutput, error) {
		configuration, err := store.CreateConfiguration(ctx, input.Body)
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerGetSantaConfiguration(api huma.API, store *configurations.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-santa-configuration",
		Method:      http.MethodGet,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaTag},
		Summary:     "Get a Santa configuration",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaConfigurationGetInput) (*santaConfigurationOutput, error) {
		id, err := parseResourceID(input.ID, santaConfigurationResource)
		if err != nil {
			return nil, err
		}
		configuration, err := store.GetConfigurationByID(ctx, id)
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerPatchSantaConfiguration(api huma.API, store *configurations.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-santa-configuration",
		Method:      http.MethodPatch,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaTag},
		Summary:     "Update a Santa configuration",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *santaConfigurationPatchInput) (*santaConfigurationOutput, error) {
		id, err := parseResourceID(input.ID, santaConfigurationResource)
		if err != nil {
			return nil, err
		}
		configuration, err := store.UpdateConfiguration(ctx, id, input.Body)
		if err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &santaConfigurationOutput{Body: *configuration}, nil
	})
}

func registerDeleteSantaConfiguration(api huma.API, store *configurations.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-configuration",
		Method:      http.MethodDelete,
		Path:        santaConfigurationIDPath,
		Tags:        []string{santaTag},
		Summary:     "Delete a Santa configuration",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaConfigurationDeleteInput) (*struct{}, error) {
		id, err := parseResourceID(input.ID, santaConfigurationResource)
		if err != nil {
			return nil, err
		}
		if err := store.DeleteConfiguration(ctx, id); err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func registerReorderSantaConfigurations(api huma.API, store *configurations.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-santa-configurations",
		Method:      http.MethodPut,
		Path:        "/api/santa/configurations/order",
		Tags:        []string{santaTag},
		Summary:     "Reorder Santa configurations",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaConfigurationReorderInput) (*struct{}, error) {
		if err := store.ReorderConfigurations(ctx, input.Body.OrderedIDs); err != nil {
			return nil, santaConfigurationMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func santaConfigurationMutationError(err error) error {
	var conflict *configurations.ConfigurationLabelConflictError
	if errors.As(err, &conflict) {
		return santaConfigurationConflictError{
			LabelID:           conflict.LabelID,
			ConfigurationID:   conflict.ConfigurationID,
			ConfigurationName: conflict.ConfigurationName,
		}
	}
	return resourceMutationError(santaConfigurationResource, err)
}

// Santa rules.

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
	Body santarules.RuleCreate
}

type santaRulePatchInput struct {
	ID   string `path:"id"`
	Body santarules.RuleUpdate
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
	Body paginatedBody[santarules.Rule]
}

type santaRuleOutput struct {
	Body santarules.Rule
}

func (input santaRuleListInput) params() santarules.RuleListParams {
	return santarules.RuleListParams{
		ListParams: dbutil.ListParams{
			Q:              input.Q,
			Page:           input.Page,
			PerPage:        input.PerPage,
			OrderKey:       input.OrderKey,
			OrderDirection: input.OrderDirection,
		},
		RuleType: santarules.RuleType(input.RuleType),
	}
}

func RegisterSantaRules(api huma.API, store *santarules.Store) {
	registerListSantaRules(api, store)
	registerCreateSantaRule(api, store)
	registerGetSantaRule(api, store)
	registerPatchSantaRule(api, store)
	registerDeleteSantaRule(api, store)
	registerReorderSantaRuleIncludes(api, store)
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
		return &santaRuleListOutput{Body: paginatedBody[santarules.Rule]{Items: rules, Count: count}}, nil
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
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusConflict},
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

func registerDeleteSantaRule(api huma.API, store *santarules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-rule",
		Method:      http.MethodDelete,
		Path:        santaRuleIDPath,
		Tags:        []string{santaTag},
		Summary:     "Delete a Santa rule",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleDeleteInput) (*struct{}, error) {
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

func registerReorderSantaRuleIncludes(api huma.API, store *santarules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-santa-rule-includes",
		Method:      http.MethodPut,
		Path:        "/api/santa/rules/{id}/includes/order",
		Tags:        []string{santaTag},
		Summary:     "Reorder Santa rule includes",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaRuleReorderIncludesInput) (*struct{}, error) {
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

// Santa events.

type santaEventListInput struct {
	HostID   int64     `query:"host_id,omitempty"`
	Decision string    `query:"decision,omitempty"`
	Since    time.Time `query:"since,omitempty"`
	Limit    int       `query:"limit,omitempty"`
	After    string    `query:"after,omitempty"`
}

type santaEventListOutput struct {
	Body santaevents.EventPage
}

func (input santaEventListInput) params() (santaevents.EventListParams, error) {
	var hostID int64
	if input.HostID != 0 {
		if input.HostID < 0 {
			return santaevents.EventListParams{}, huma.Error400BadRequest("host_id must be positive")
		}
		hostID = input.HostID
	}
	var since *time.Time
	if !input.Since.IsZero() {
		since = &input.Since
	}
	return santaevents.EventListParams{
		HostID:   hostID,
		Decision: santaevents.DecisionFilter(input.Decision),
		Since:    since,
		Limit:    input.Limit,
		After:    input.After,
	}, nil
}

func RegisterSantaEvents(api huma.API, store *santaevents.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-events",
		Method:      http.MethodGet,
		Path:        "/api/santa/events",
		Tags:        []string{santaTag},
		Summary:     "List Santa execution events",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *santaEventListInput) (*santaEventListOutput, error) {
		params, err := input.params()
		if err != nil {
			return nil, err
		}
		page, err := store.ListEvents(ctx, params)
		if err != nil {
			return nil, resourceMutationError("Santa event", err)
		}
		return &santaEventListOutput{Body: page}, nil
	})
}

// Santa sync tokens.

type santaSyncTokenDeleteInput struct {
	ID string `path:"id"`
}

type santaSyncTokenListOutput struct {
	Body []santasync.SyncToken
}

type santaSyncTokenCreateOutput struct {
	Body santasync.SyncToken
}

func RegisterSantaSyncTokens(api huma.API, store *santasync.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-sync-tokens",
		Method:      http.MethodGet,
		Path:        "/api/santa/sync-tokens",
		Tags:        []string{santaTag},
		Summary:     "List Santa sync tokens",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*santaSyncTokenListOutput, error) {
		tokens, err := store.ListTokens(ctx)
		if err != nil {
			return nil, err
		}
		return &santaSyncTokenListOutput{Body: tokens}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-santa-sync-token",
		Method:        http.MethodPost,
		Path:          "/api/santa/sync-tokens",
		Tags:          []string{santaTag},
		Summary:       "Create Santa sync token",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*santaSyncTokenCreateOutput, error) {
		token, err := store.CreateToken(ctx)
		if err != nil {
			return nil, err
		}
		return &santaSyncTokenCreateOutput{Body: token}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-sync-token",
		Method:      http.MethodDelete,
		Path:        "/api/santa/sync-tokens/{id}",
		Tags:        []string{santaTag},
		Summary:     "Delete Santa sync token",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaSyncTokenDeleteInput) (*struct{}, error) {
		id, err := parseResourceID(input.ID, "sync token")
		if err != nil {
			return nil, err
		}
		err = store.DeleteToken(ctx, id)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("sync token not found")
		}
		if err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

// Host subresource: effective Santa rules for a host.

type hostSantaEffectiveRulesOutput struct {
	Body paginatedBody[santarules.EffectiveRuleStatus]
}

type hostSantaEffectiveRulesInput struct {
	ID             string `path:"id"`
	Page           int    `query:"page,omitempty"`
	PerPage        int    `query:"per_page,omitempty"`
	OrderKey       string `query:"order_key,omitempty"`
	OrderDirection string `query:"order_direction,omitempty"`
}

// RegisterHostSantaEffectiveRules registers the Santa effective-rules host subresource.
func RegisterHostSantaEffectiveRules(api huma.API, hostStore *hosts.Store, santaRuleStore *santarules.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-santa-effective-rules",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/santa/effective-rules",
		Tags:        []string{hostsTag},
		Summary:     "List effective Santa rules for a host",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostSantaEffectiveRulesInput) (*hostSantaEffectiveRulesOutput, error) {
		id, err := parseResourceID(input.ID, hostResource)
		if err != nil {
			return nil, err
		}
		if hostStore == nil || santaRuleStore == nil {
			return nil, huma.Error404NotFound("host not found")
		}
		if _, err := hostStore.GetByID(ctx, id); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		rows, count, err := santaRuleStore.ListEffectiveRulesForHost(ctx, id, santarules.EffectiveRuleListParams{
			ListParams: dbutil.ListParams{
				Page:           input.Page,
				PerPage:        input.PerPage,
				OrderKey:       input.OrderKey,
				OrderDirection: input.OrderDirection,
			},
		})
		if err != nil {
			return nil, resourceMutationError("Santa effective rule", err)
		}
		return &hostSantaEffectiveRulesOutput{
			Body: paginatedBody[santarules.EffectiveRuleStatus]{Items: rows, Count: count},
		}, nil
	})
}
