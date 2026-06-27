package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
)

type hostSantaRulesOutput struct {
	Body apitypes.Page[rules.RuleStatus]
}

type hostSantaRulesInput struct {
	apitypes.ListQueryInput

	HostID int64 `path:"id"`
}

func registerHostSantaRules(api huma.API, ruleStore *rules.Store, hostStore *hosts.Store) {
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
			return nil, err
		}
		rows, count, err := ruleStore.ListRuleStatusesForHost(ctx, input.HostID, rules.RuleStatusListParams{
			ListParams: input.ListQueryInput.Params(),
		})
		if err != nil {
			return nil, apitypes.ResourceMutationError(santaRuleResource, err)
		}
		return &hostSantaRulesOutput{
			Body: apitypes.Page[rules.RuleStatus]{Items: rows, Count: int32(count)},
		}, nil
	})
}
