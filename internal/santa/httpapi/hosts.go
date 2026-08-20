package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
)

type hostSantaStateInput struct {
	ID int64 `path:"id"`
}

type hostSantaStateOutput struct {
	Body santa.HostState
}

func registerHostSantaState(
	humaAPI huma.API,
	store *santa.HostStateService,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-host-santa-state",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/santa",
		Tags:        []string{api.TagHosts},
		Summary:     "Get Santa state for a host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostSantaStateInput) (*hostSantaStateOutput, error) {
		state, err := store.LoadHostState(ctx, input.ID)
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"get-host-santa-state",
				err,
				"host_id", input.ID,
			)
		}
		if state == nil {
			return nil, huma.Error404NotFound("")
		}
		return &hostSantaStateOutput{Body: *state}, nil
	})
}

type hostSantaRulesInput struct {
	api.ListQueryInput

	ID int64 `path:"id"`
}

type hostSantaRulesOutput struct {
	Body api.Page[rules.RuleStatus]
}

func registerHostSantaRules(
	humaAPI huma.API,
	configurationStore *configurations.Store,
	ruleStore *rules.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-host-santa-rules",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/santa/rules",
		Tags:        []string{api.TagHosts},
		Summary:     "List Santa rules for a host",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *hostSantaRulesInput) (*hostSantaRulesOutput, error) {
		configuration, err := configurationStore.ResolveConfigurationForHost(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"list-host-santa-rules",
				"host",
				err,
				"host_id", input.ID,
			)
		}
		var configurationID int64
		if configuration != nil {
			configurationID = configuration.ID
		}
		rows, count, err := ruleStore.ListRuleStatusesForHost(ctx, input.ID, configurationID, rules.RuleStatusListParams{
			ListParams: input.Params(),
		})
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"list-host-santa-rules",
				"host",
				err,
				"host_id", input.ID,
			)
		}
		return &hostSantaRulesOutput{
			Body: api.Page[rules.RuleStatus]{Items: rows, Count: count},
		}, nil
	})
}
