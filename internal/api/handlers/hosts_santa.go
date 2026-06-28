package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
)

type hostSantaStateLoader interface {
	LoadHostState(context.Context, int64) (*santa.HostState, error)
}

type hostSantaRulesOutput struct {
	Body Page[rules.RuleStatus]
}

type hostSantaRulesInput struct {
	ListQueryInput

	ID int64 `path:"id"`
}

func registerHostSantaState(
	api huma.API,
	store hostSantaStateLoader,
	logger *slog.Logger,
) {
	registerHostState(
		api,
		"get-host-santa-state",
		"/api/hosts/{id}/santa",
		"Get Santa state for a host",
		"santa state not found",
		store.LoadHostState,
		logger,
	)
}

func registerHostSantaRules(api huma.API, ruleStore *rules.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-santa-rules",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/santa/rules",
		Tags:        []string{hostsTag},
		Summary:     "List Santa rules for a host",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostSantaRulesInput) (*hostSantaRulesOutput, error) {
		rows, count, err := ruleStore.ListRuleStatusesForHost(ctx, input.ID, rules.RuleStatusListParams{
			ListParams: input.ListQueryInput.params(),
		})
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"list-host-santa-rules",
				hostResource,
				err,
				"host_id", input.ID,
			)
		}
		return &hostSantaRulesOutput{
			Body: Page[rules.RuleStatus]{Items: rows, Count: count},
		}, nil
	})
}
