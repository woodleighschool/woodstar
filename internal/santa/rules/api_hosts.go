package rules

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

const hostsTag = "Hosts"

type hostSantaRulesOutput struct {
	Body apitypes.Page[RuleStatus]
}

type hostSantaRulesInput struct {
	apitypes.ListQueryInput

	HostID int64 `path:"id"`
}

func RegisterHostAdminRoutes(api huma.API, ruleStore *Store, hostStore *hosts.Store) {
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
		rows, count, err := ruleStore.ListRuleStatusesForHost(ctx, input.HostID, RuleStatusListParams{
			ListParams: input.ListQueryInput.Params(),
		})
		if err != nil {
			return nil, apitypes.ResourceMutationError(santaRuleResource, err)
		}
		return &hostSantaRulesOutput{
			Body: apitypes.Page[RuleStatus]{Items: rows, Count: count},
		}, nil
	})
}
