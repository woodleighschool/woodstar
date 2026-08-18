package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/osquery/history"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
)

const defaultHistoryWindow = 24 * time.Hour

type historyInput struct {
	Since time.Time `query:"since,omitempty"`
}

func (input historyInput) since() time.Time {
	if input.Since.IsZero() {
		return time.Now().Add(-defaultHistoryWindow)
	}
	return input.Since
}

type policyHistoryInput struct {
	ID    int64     `path:"id"`
	Since time.Time `query:"since,omitempty"`
}

func (input policyHistoryInput) since() time.Time {
	return historyInput{Since: input.Since}.since()
}

type hostHistoryOutput struct {
	Body []history.HostStatusPoint
}

type policyHistoryOutput struct {
	Body []history.PolicyStatusPoint
}

func registerHistory(
	humaAPI huma.API,
	historyStore *history.Store,
	policyStore *policies.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-osquery-host-status-history",
		Method:      http.MethodGet,
		Path:        "/api/osquery/host-status-history",
		Tags:        []string{api.TagOsqueryOverview},
		Summary:     "List host status history",
	}, func(ctx context.Context, input *historyInput) (*hostHistoryOutput, error) {
		points, err := historyStore.ListHostStatus(ctx, input.since())
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "list-osquery-host-status-history", err)
		}
		return &hostHistoryOutput{Body: points}, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-osquery-policy-status-history",
		Method:      http.MethodGet,
		Path:        "/api/osquery/policies/{id}/status-history",
		Tags:        []string{api.TagOsqueryPolicies},
		Summary:     "List policy status history",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *policyHistoryInput) (*policyHistoryOutput, error) {
		if _, err := policyStore.GetByID(ctx, input.ID); err != nil {
			return nil, api.ResourceError(
				ctx, logger, "list-osquery-policy-status-history", policyResource, err, "id", input.ID,
			)
		}
		points, err := historyStore.ListPolicyStatus(ctx, input.ID, input.since())
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "list-osquery-policy-status-history", err)
		}
		return &policyHistoryOutput{Body: points}, nil
	})
}
