package handlers

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
)

type hostSantaStateLoader interface {
	LoadHostState(ctx context.Context, hostID int64) (*santa.HostState, error)
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
		store.LoadHostState,
		logger,
	)
}

func registerHostSantaRules(api huma.API, ruleStore *rules.Store, logger *slog.Logger) {
	registerHostPage(
		api,
		"list-host-santa-rules",
		"/api/hosts/{id}/santa/rules",
		"List Santa rules for a host",
		func(
			ctx context.Context,
			hostID int64,
			params dbutil.ListParams,
		) ([]rules.RuleStatus, int, error) {
			return ruleStore.ListRuleStatusesForHost(ctx, hostID, rules.RuleStatusListParams{
				ListParams: params,
			})
		},
		logger,
	)
}
