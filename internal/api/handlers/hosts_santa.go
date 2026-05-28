package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
)

type santaHostDetailContributor struct {
	loader santaHostStateLoader
}

type santaHostStateLoader interface {
	LoadHostState(context.Context, int64) (*santa.HostState, error)
}

// SantaHostDetailContributor returns a host detail contributor backed by Santa state.
func SantaHostDetailContributor(loader santaHostStateLoader) HostDetailContributor {
	if loader == nil {
		return nil
	}
	return santaHostDetailContributor{loader: loader}
}

func (c santaHostDetailContributor) ContributeHostDetail(
	ctx context.Context,
	hostID int64,
	body *hostDetailBody,
) error {
	detail, err := c.loader.LoadHostState(ctx, hostID)
	if err != nil {
		return err
	}
	body.Santa = detail
	return nil
}

type hostSantaEffectiveRulesOutput struct {
	Body paginatedBody[santarules.EffectiveRuleStatus]
}

type hostSantaEffectiveRulesInput struct {
	ID int64 `path:"id"`
	ListQueryInput
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
		if hostStore == nil || santaRuleStore == nil {
			return nil, huma.Error404NotFound("host not found")
		}
		if _, err := hostStore.GetByID(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		rows, count, err := santaRuleStore.ListEffectiveRulesForHost(ctx, input.ID, santarules.EffectiveRuleListParams{
			ListParams: input.ListQueryInput.params(),
		})
		if err != nil {
			return nil, resourceMutationError("Santa effective rule", err)
		}
		return &hostSantaEffectiveRulesOutput{
			Body: paginatedBody[santarules.EffectiveRuleStatus]{Items: rows, Count: count},
		}, nil
	})
}
