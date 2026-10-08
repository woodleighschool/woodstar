import {
  keepPreviousData,
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import type {
  ApiError,
  OsqueryPolicy,
  OsqueryPolicyHostStatus,
  OsqueryPolicyMutation,
  OsqueryPolicyRemediationBatchSummary,
  OsqueryPolicyRemediationRun,
  OsqueryPolicyRemediationSource,
  PagePolicy,
  PagePolicyHostStatus,
} from "@lib/api";
import {
  bulkDeleteOsqueryPolicies,
  createOsqueryPolicy,
  deleteOsqueryPolicy,
  getOsqueryPolicy,
  getOsqueryPolicyRemediationRun,
  getOsqueryPolicyRemediationSource,
  listOsqueryPolicyResults,
  listOsqueryPolicies,
  nullOn404,
  runOsqueryPolicyRemediations,
  unwrap,
  updateOsqueryPolicy,
} from "@lib/api";
import type {
  ListOsqueryPolicyResultsData,
  ListOsqueryPoliciesData,
} from "@lib/api-client/types.gen";
import { baseListParams, collectAllPages } from "@lib/pagination";
import { invalidateAfterDelete } from "@lib/query-invalidation";
import { countLabel } from "@lib/utils";

type QueryParams = Record<string, unknown>;

type PolicyListParams = NonNullable<ListOsqueryPoliciesData["query"]>;
type PolicyResultsParams = NonNullable<ListOsqueryPolicyResultsData["query"]>;
type EnabledOptions = { enabled?: boolean };

const POLICY_REFRESH_MS = 30_000;

const policyKeys = {
  all: ["osquery", "policies"] as const,
  list: (params?: QueryParams) => ["osquery", "policies", "list", params ?? {}] as const,
  detail: (id: number) => ["osquery", "policies", "detail", id] as const,
  remediationSource: (id: number) =>
    ["osquery", "policies", "detail", id, "remediation-source"] as const,
  remediationRun: (id: number, hostID: number | null) =>
    ["osquery", "policies", "detail", id, "remediation", hostID] as const,
  resultsRoot: (id: number) => ["osquery", "policies", "detail", id, "results"] as const,
  results: (id: number, params?: QueryParams) =>
    [...policyKeys.resultsRoot(id), params ?? {}] as const,
};

export function policyQueryOptions(id: number) {
  return queryOptions<OsqueryPolicy, ApiError>({
    queryKey: policyKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getOsqueryPolicy({ path: { id }, signal })),
  });
}

export function usePolicies(params: PolicyListParams = {}) {
  const queryParams = baseListParams(params);

  return useQuery<PagePolicy, ApiError>({
    queryKey: policyKeys.list(queryParams),
    queryFn: ({ signal }) => unwrap(listOsqueryPolicies({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: POLICY_REFRESH_MS,
  });
}

export function usePolicy(id: number) {
  return useQuery(policyQueryOptions(id));
}

export function usePolicyRemediationSource(id: number, options: EnabledOptions = {}) {
  return useQuery<OsqueryPolicyRemediationSource, ApiError>({
    queryKey: policyKeys.remediationSource(id),
    queryFn: ({ signal }) => unwrap(getOsqueryPolicyRemediationSource({ path: { id }, signal })),
    enabled: options.enabled,
  });
}

export function usePolicyRemediationRun(policyID: number, hostID: number | null) {
  return useQuery<OsqueryPolicyRemediationRun | null, ApiError>({
    queryKey: policyKeys.remediationRun(policyID, hostID),
    queryFn: ({ signal }) =>
      nullOn404(
        getOsqueryPolicyRemediationRun({
          path: { id: policyID, host_id: hostID ?? 0 },
          signal,
        }),
      ),
    enabled: hostID !== null,
    refetchInterval: POLICY_REFRESH_MS,
  });
}

export function usePolicyResults(
  id: number,
  params: PolicyResultsParams = {},
  options: EnabledOptions = {},
) {
  const queryParams = policyResultQueryParams(params);
  return useQuery<PagePolicyHostStatus, ApiError>({
    queryKey: policyKeys.results(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listOsqueryPolicyResults({
          path: { id },
          query: queryParams,
          signal,
        }),
      ),
    enabled: options.enabled,
    placeholderData: keepPreviousData,
    refetchInterval: POLICY_REFRESH_MS,
  });
}

export function listAllPolicyResults(
  id: number,
  params: PolicyResultsParams = {},
): Promise<OsqueryPolicyHostStatus[]> {
  return collectAllPages((page, perPage) =>
    unwrap(
      listOsqueryPolicyResults({
        path: { id },
        query: policyResultQueryParams({ ...params, page, per_page: perPage }),
      }),
    ),
  );
}

function policyResultQueryParams(params: PolicyResultsParams) {
  return {
    ...baseListParams(params),
    status: params.status,
    remediation: params.remediation,
  };
}

export function useCreatePolicy() {
  const queryClient = useQueryClient();
  return useMutation<OsqueryPolicy, ApiError, OsqueryPolicyMutation>({
    mutationFn: (body) => unwrap(createOsqueryPolicy({ body })),
    onSuccess: async () => {
      toast.add({ title: "Policy Created", type: "success" });
      await queryClient.invalidateQueries({ queryKey: policyKeys.all });
    },
  });
}

export function useUpdatePolicy(id: number) {
  const queryClient = useQueryClient();
  return useMutation<OsqueryPolicy, ApiError, OsqueryPolicyMutation>({
    mutationFn: (body) =>
      unwrap(
        updateOsqueryPolicy({
          path: { id },
          body,
        }),
      ),
    onSuccess: async (saved) => {
      const previous = queryClient.getQueryData<OsqueryPolicy>(policyKeys.detail(id));
      const queryChanged = previous !== undefined && previous.query !== saved.query;
      if (queryChanged) {
        queryClient.removeQueries({ queryKey: policyKeys.resultsRoot(id) });
      }
      toast.add({
        title: queryChanged ? "Policy saved and results cleared" : "Policy Saved",
        type: "success",
      });
      await queryClient.invalidateQueries({ queryKey: policyKeys.all });
    },
  });
}

export function useDeletePolicy() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteOsqueryPolicy({ path: { id } })),
    onSuccess: (_, id) =>
      invalidateAfterDelete(queryClient, policyKeys.detail(id), [policyKeys.all]),
  });
}

export function useBulkDeletePolicies() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteOsqueryPolicies({ query: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: policyKeys.all });
    },
  });
}

type PolicyHostsMutation =
  | { policyID: number; hostIDs: number[]; allFailures?: never }
  | { policyID: number; hostIDs?: never; allFailures: true };

export function useRunPolicyRemediations() {
  const queryClient = useQueryClient();
  return useMutation<OsqueryPolicyRemediationBatchSummary, ApiError, PolicyHostsMutation>({
    mutationFn: ({ policyID, hostIDs, allFailures }) =>
      unwrap(
        runOsqueryPolicyRemediations({
          path: { id: policyID },
          query: hostIDs ? { host_ids: hostIDs } : { all_failures: allFailures },
        }),
      ),
    onSuccess: async (summary, { policyID }) => {
      toast.add({
        title:
          summary.queued === 0
            ? "No Remediation Queued"
            : `Remediation Queued for ${countLabel(summary.queued, "Host")}`,
        description:
          summary.skipped > 0
            ? `${countLabel(summary.skipped, "host was", "hosts were")} skipped because it was no longer failing, remediation was unavailable, or remediation was already active.`
            : undefined,
        type: summary.queued === 0 ? "info" : "success",
      });
      await queryClient.invalidateQueries({ queryKey: policyKeys.resultsRoot(policyID) });
    },
    onError: (error) => {
      toast.add({
        title: "Remediation could not be queued",
        description: error.message,
        type: "error",
      });
    },
  });
}
