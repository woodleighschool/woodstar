package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
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

	HostID int64 `path:"id"`
}

func registerHostSantaState(
	api huma.API,
	store hostSantaStateLoader,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	registerHostState(
		api,
		"get-host-santa-state",
		"/api/hosts/{id}/santa",
		"Get Santa state for a host",
		"santa state not found",
		hostStore,
		store.LoadHostState,
		logger,
	)
}

func registerHostSantaRules(api huma.API, ruleStore *rules.Store, hostStore *hosts.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-santa-rules",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/santa/rules",
		Tags:        []string{hostsTag},
		Summary:     "List Santa rules for a host",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostSantaRulesInput) (*hostSantaRulesOutput, error) {
		if hostStore == nil || ruleStore == nil {
			return nil, huma.Error404NotFound("host not found")
		}
		if _, err := hostStore.GetByID(ctx, input.HostID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, handlerError(ctx, logger, "list-host-santa-rules", err, "host_id", input.HostID)
		}
		rows, count, err := ruleStore.ListRuleStatusesForHost(ctx, input.HostID, rules.RuleStatusListParams{
			ListParams: input.ListQueryInput.Params(),
		})
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"list-host-santa-rules",
				santaRuleResource,
				err,
				"host_id", input.HostID,
			)
		}
		return &hostSantaRulesOutput{
			Body: Page[rules.RuleStatus]{Items: rows, Count: int32(count)},
		}, nil
	})
}
