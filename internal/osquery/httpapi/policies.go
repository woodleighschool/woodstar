package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/goodies/auth/authn"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
)

const (
	policyResource = "policy"
	policyIDPath   = "/api/osquery/policies/{id}"
)

type policyListInput struct {
	api.ListQueryInput
}

func (input policyListInput) params() policies.PolicyListParams {
	return policies.PolicyListParams{
		ListParams: input.Params(),
	}
}

type policyGetInput struct {
	ID int64 `path:"id"`
}

type policyResultsInput struct {
	api.ListQueryInput

	ID          int64                                    `path:"id"`
	Status      []policies.PolicyStatus                  `          query:"status,omitempty"`
	Remediation []policies.PolicyRemediationStatusFilter `          query:"remediation,omitempty"`
}

func (input policyResultsInput) params() policies.PolicyResultListParams {
	return policies.PolicyResultListParams{
		ListParams:          input.Params(),
		Statuses:            input.Status,
		RemediationStatuses: input.Remediation,
	}
}

type policyCreateInput struct {
	Body policies.PolicyMutation
}

type policyPutInput struct {
	ID   int64 `path:"id"`
	Body policies.PolicyMutation
}

type policyDeleteInput struct {
	ID int64 `path:"id"`
}

type policyHostInput struct {
	ID     int64 `path:"id"`
	HostID int64 `path:"host_id"`
}

type policyRemediationsInput struct {
	ID          int64   `path:"id"`
	HostIDs     []int64 `query:"host_ids,omitempty"`
	AllFailures bool    `query:"all_failures,omitempty"`
}

type policyListOutput struct {
	Body api.Page[policies.Policy]
}

type policyOutput struct {
	Body policies.Policy
}

type policyResultsOutput struct {
	Body api.Page[policies.PolicyHostStatus]
}

type policyRemediationSourceOutput struct {
	Body policies.PolicyRemediationSource
}

type policyRemediationRunOutput struct {
	Body policies.PolicyRemediationRun
}

type policyRemediationBatchSummaryOutput struct {
	Body policies.PolicyRemediationBatchSummary
}

func registerOsqueryPolicies(
	humaAPI huma.API,
	remediationsAPI huma.API,
	policyStore *policies.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	registerListPolicies(humaAPI, policyStore, logger)
	registerCreatePolicy(humaAPI, policyStore, activityRecorder, logger)
	registerGetPolicy(humaAPI, policyStore, logger)
	registerUpdatePolicy(humaAPI, policyStore, activityRecorder, logger)
	registerDeletePolicy(humaAPI, policyStore, activityRecorder, logger)
	registerBulkDeletePolicies(humaAPI, policyStore, activityRecorder, logger)
	registerPolicyResults(humaAPI, policyStore, logger)
	registerRunPolicyRemediations(remediationsAPI, policyStore, activityRecorder, logger)
	registerPolicyRemediationSource(remediationsAPI, policyStore, logger)
	registerPolicyRemediationRun(remediationsAPI, policyStore, logger)
}

func registerListPolicies(humaAPI huma.API, policyStore *policies.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-osquery-policies",
		Method:      http.MethodGet,
		Path:        "/api/osquery/policies",
		Tags:        []string{api.TagOsqueryPolicies},
		Summary:     "List policies",
	}, func(ctx context.Context, input *policyListInput) (*policyListOutput, error) {
		items, count, err := policyStore.List(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-osquery-policies", policyResource, err)
		}
		return &policyListOutput{Body: api.Page[policies.Policy]{Items: items, Count: count}}, nil
	})
}

func registerRunPolicyRemediations(
	humaAPI huma.API,
	policyStore *policies.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "run-osquery-policy-remediations",
		Method:        http.MethodPost,
		Path:          "/api/osquery/policies/{id}/remediation",
		Tags:          []string{api.TagOsqueryPolicies},
		Summary:       "Run policy remediations",
		DefaultStatus: http.StatusAccepted,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *policyRemediationsInput) (*policyRemediationBatchSummaryOutput, error) {
		policy, err := policyStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(
				ctx, logger, "run-osquery-policy-remediations", policyResource, err, "id", input.ID,
			)
		}
		summary, err := policyStore.RunRemediations(
			ctx,
			input.ID,
			input.HostIDs,
			input.AllFailures,
		)
		if err != nil {
			return nil, api.ResourceError(
				ctx, logger, "run-osquery-policy-remediations", policyResource, err,
				"id", input.ID, "host_ids", input.HostIDs, "all_failures", input.AllFailures,
			)
		}
		activity.RecordUser(
			ctx,
			activityRecorder,
			logger,
			activity.AreaOsquery,
			activity.ActionPolicyRemediationRequested,
			activity.Resource(policyResource, policy.ID, policy.Name),
		)
		return &policyRemediationBatchSummaryOutput{Body: *summary}, nil
	})
}

func registerPolicyRemediationSource(
	humaAPI huma.API,
	policyStore *policies.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-osquery-policy-remediation-source",
		Method:      http.MethodGet,
		Path:        "/api/osquery/policies/{id}/remediation",
		Tags:        []string{api.TagOsqueryPolicies},
		Summary:     "Get policy remediation source",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *policyGetInput) (*policyRemediationSourceOutput, error) {
		source, err := policyStore.RemediationSource(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(
				ctx, logger, "get-osquery-policy-remediation-source", policyResource, err,
				"id", input.ID,
			)
		}
		return &policyRemediationSourceOutput{Body: *source}, nil
	})
}

func registerPolicyRemediationRun(
	humaAPI huma.API,
	policyStore *policies.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-osquery-policy-remediation-run",
		Method:      http.MethodGet,
		Path:        "/api/osquery/policies/{id}/hosts/{host_id}/remediation",
		Tags:        []string{api.TagOsqueryPolicies},
		Summary:     "Get current policy remediation run",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *policyHostInput) (*policyRemediationRunOutput, error) {
		run, err := policyStore.RemediationRun(ctx, input.ID, input.HostID)
		if err != nil {
			return nil, api.ResourceError(
				ctx, logger, "get-osquery-policy-remediation-run", policyResource, err,
				"id", input.ID, "host_id", input.HostID,
			)
		}
		return &policyRemediationRunOutput{Body: *run}, nil
	})
}

func registerCreatePolicy(
	humaAPI huma.API,
	policyStore *policies.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-osquery-policy",
		Method:        http.MethodPost,
		Path:          "/api/osquery/policies",
		Tags:          []string{api.TagOsqueryPolicies},
		Summary:       "Create a policy",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *policyCreateInput) (*policyOutput, error) {
		policy, err := policyStore.Create(ctx, policies.PolicyCreateMutation{
			PolicyMutation:  input.Body,
			CreatedByUserID: authn.CurrentPrincipalID(ctx),
		})
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-osquery-policy", policyResource, err)
		}
		activity.RecordUser(ctx, activityRecorder, logger, activity.AreaOsquery, activity.ActionPolicyCreated,
			activity.Resource(policyResource, policy.ID, policy.Name))
		return &policyOutput{Body: *policy}, nil
	})
}

func registerGetPolicy(humaAPI huma.API, policyStore *policies.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-osquery-policy",
		Method:      http.MethodGet,
		Path:        policyIDPath,
		Tags:        []string{api.TagOsqueryPolicies},
		Summary:     "Get a policy",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *policyGetInput) (*policyOutput, error) {
		policy, err := policyStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-osquery-policy", policyResource, err, "id", input.ID)
		}
		return &policyOutput{Body: *policy}, nil
	})
}

//nolint:dupl // Policy and report handlers intentionally keep their domain contracts explicit.
func registerUpdatePolicy(
	humaAPI huma.API,
	policyStore *policies.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-osquery-policy",
		Method:      http.MethodPut,
		Path:        policyIDPath,
		Tags:        []string{api.TagOsqueryPolicies},
		Summary:     "Update a policy",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *policyPutInput) (*policyOutput, error) {
		policy, err := policyStore.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"update-osquery-policy",
				policyResource,
				err,
				"id",
				input.ID,
			)
		}
		activity.RecordUser(ctx, activityRecorder, logger, activity.AreaOsquery, activity.ActionPolicyUpdated,
			activity.Resource(policyResource, policy.ID, policy.Name))
		return &policyOutput{Body: *policy}, nil
	})
}

func registerDeletePolicy(
	humaAPI huma.API,
	policyStore *policies.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-osquery-policy",
		Method:      http.MethodDelete,
		Path:        policyIDPath,
		Tags:        []string{api.TagOsqueryPolicies},
		Summary:     "Delete a policy",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *policyDeleteInput) (*struct{}, error) {
		policy, err := policyStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-osquery-policy", policyResource, err, "id", input.ID)
		}
		if err := policyStore.Delete(ctx, input.ID); err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"delete-osquery-policy",
				policyResource,
				err,
				"id",
				input.ID,
			)
		}
		activity.RecordUser(ctx, activityRecorder, logger, activity.AreaOsquery, activity.ActionPolicyDeleted,
			activity.Resource(policyResource, policy.ID, policy.Name))
		return &struct{}{}, nil
	})
}

func registerBulkDeletePolicies(
	humaAPI huma.API,
	policyStore *policies.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "bulk-delete-osquery-policies",
		Method:        http.MethodDelete,
		Path:          "/api/osquery/policies",
		Tags:          []string{api.TagOsqueryPolicies},
		Summary:       "Delete policies",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *api.DeleteManyInput) (*struct{}, error) {
		deleted, err := policyStore.DeleteMany(ctx, input.IDs)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "bulk-delete-osquery-policies", err)
		}
		if deleted > 0 {
			activity.RecordUser(ctx, activityRecorder, logger, activity.AreaOsquery, activity.ActionPoliciesDeleted,
				activity.Collection(policyResource, fmt.Sprintf("%d policies", deleted)))
		}
		return &struct{}{}, nil
	})
}

func registerPolicyResults(humaAPI huma.API, policyStore *policies.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-osquery-policy-results",
		Method:      http.MethodGet,
		Path:        "/api/osquery/policies/{id}/results",
		Tags:        []string{api.TagOsqueryPolicies},
		Summary:     "List policy results",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *policyResultsInput) (*policyResultsOutput, error) {
		rows, count, err := policyStore.PolicyResults(ctx, input.ID, input.params())
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"list-osquery-policy-results",
				policyResource,
				err,
				"id",
				input.ID,
			)
		}
		return &policyResultsOutput{
			Body: api.Page[policies.PolicyHostStatus]{Items: rows, Count: count},
		}, nil
	})
}
