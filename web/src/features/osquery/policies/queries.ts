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
  OsqueryPolicyRemediationRun,
  OsqueryPolicyRemediationRunSummary,
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
  resetOsqueryPolicyResult,
  runOsqueryPolicyRemediation,
  unwrap,
  updateOsqueryPolicy,
} from "@lib/api";
import type {
  ListOsqueryPolicyResultsData,
  ListOsqueryPoliciesData,
} from "@lib/api-client/types.gen";
import { baseListParams, collectAllPages } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

type QueryParams = Record<string, unknown>;

type PolicyListParams = NonNullable<ListOsqueryPoliciesData["query"]>;
type PolicyResultsParams = NonNullable<ListOsqueryPolicyResultsData["query"]>;

const POLICY_REFRESH_MS = 30_000;

const policyKeys = {
  all: ["osquery", "policies"] as const,
  list: (params?: QueryParams) => ["osquery", "policies", "list", params ?? {}] as const,
  detail: (id: number | null) => ["osquery", "policies", "detail", id] as const,
  remediationSource: (id: number | null) =>
    ["osquery", "policies", "detail", id, "remediation-source"] as const,
  remediationRun: (id: number | null, hostID: number | null) =>
    ["osquery", "policies", "detail", id, "remediation", hostID] as const,
  resultsRoot: (id: number | null) => ["osquery", "policies", "detail", id, "results"] as const,
  results: (id: number | null, params?: QueryParams) =>
    [...policyKeys.resultsRoot(id), params ?? {}] as const,
};

export function policyQueryOptions(id: number | null) {
  return queryOptions<OsqueryPolicy, ApiError>({
    queryKey: policyKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getOsqueryPolicy({ path: detailPath(id), signal })),
    enabled: id !== null,
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

export function usePolicy(id: number | null) {
  return useQuery(policyQueryOptions(id));
}

export function usePolicyRemediationSource(id: number | null) {
  return useQuery<OsqueryPolicyRemediationSource, ApiError>({
    queryKey: policyKeys.remediationSource(id),
    queryFn: ({ signal }) =>
      unwrap(getOsqueryPolicyRemediationSource({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function usePolicyRemediationRun(policyID: number | null, hostID: number | null) {
  return useQuery<OsqueryPolicyRemediationRun | null, ApiError>({
    queryKey: policyKeys.remediationRun(policyID, hostID),
    queryFn: ({ signal }) =>
      nullOn404(
        getOsqueryPolicyRemediationRun({
          path: { id: policyID ?? 0, host_id: hostID ?? 0 },
          signal,
        }),
      ),
    enabled: policyID !== null && hostID !== null,
    refetchInterval: POLICY_REFRESH_MS,
  });
}

export function usePolicyResults(id: number | null, params: PolicyResultsParams = {}) {
  const queryParams = policyResultQueryParams(params);
  return useQuery<PagePolicyHostStatus, ApiError>({
    queryKey: policyKeys.results(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listOsqueryPolicyResults({
          path: detailPath(id),
          query: queryParams,
          signal,
        }),
      ),
    enabled: id !== null,
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
  };
}

export function useCreatePolicy() {
  const queryClient = useQueryClient();
  return useMutation<OsqueryPolicy, ApiError, OsqueryPolicyMutation>({
    mutationFn: (body) => unwrap(createOsqueryPolicy({ body })),
    onSuccess: async () => {
      toast.add({ title: "Policy created", type: "success" });
      await queryClient.invalidateQueries({ queryKey: policyKeys.all });
    },
  });
}

export function useUpdatePolicy(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<OsqueryPolicy, ApiError, OsqueryPolicyMutation>({
    mutationFn: (body) =>
      unwrap(
        updateOsqueryPolicy({
          path: detailPath(id),
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
        title: queryChanged ? "Policy saved and results cleared" : "Policy saved",
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
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: policyKeys.all });
    },
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

type PolicyHostMutation = { policyID: number; hostID: number };

export function useResetPolicyResult() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, PolicyHostMutation>({
    mutationFn: ({ policyID, hostID }) =>
      unwrap(resetOsqueryPolicyResult({ path: { id: policyID, host_id: hostID } })),
    onSuccess: async (_, { policyID }) => {
      toast.add({ title: "Policy result reset", type: "success" });
      await queryClient.invalidateQueries({ queryKey: policyKeys.resultsRoot(policyID) });
    },
    onError: (error) => {
      toast.add({
        title: "Policy result could not be reset",
        description: error.message,
        type: "error",
      });
    },
  });
}

export function useRunPolicyRemediation() {
  const queryClient = useQueryClient();
  return useMutation<OsqueryPolicyRemediationRunSummary, ApiError, PolicyHostMutation>({
    mutationFn: ({ policyID, hostID }) =>
      unwrap(runOsqueryPolicyRemediation({ path: { id: policyID, host_id: hostID } })),
    onSuccess: async (_, { policyID, hostID }) => {
      toast.add({ title: "Remediation queued", type: "success" });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: policyKeys.resultsRoot(policyID) }),
        queryClient.invalidateQueries({ queryKey: policyKeys.remediationRun(policyID, hostID) }),
      ]);
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
