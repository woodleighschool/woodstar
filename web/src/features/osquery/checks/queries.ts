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
  OsqueryCheck,
  OsqueryCheckHostStatus,
  OsqueryCheckMutation,
  PageCheck,
  PageCheckHostStatus,
} from "@lib/api";
import {
  bulkDeleteOsqueryChecks,
  createOsqueryCheck,
  deleteOsqueryCheck,
  getOsqueryCheck,
  listOsqueryCheckResults,
  listOsqueryChecks,
  unwrap,
  updateOsqueryCheck,
} from "@lib/api";
import type { ListOsqueryCheckResultsData, ListOsqueryChecksData } from "@lib/api-client/types.gen";
import { baseListParams, collectAllPages } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

type QueryParams = Record<string, unknown>;

type CheckListParams = NonNullable<ListOsqueryChecksData["query"]>;
type CheckResultsParams = NonNullable<ListOsqueryCheckResultsData["query"]>;

const CHECK_REFRESH_MS = 30_000;

const checkKeys = {
  all: ["osquery", "checks"] as const,
  list: (params?: QueryParams) => ["osquery", "checks", "list", params ?? {}] as const,
  detail: (id: number | null) => ["osquery", "checks", "detail", id] as const,
  resultsRoot: (id: number | null) => ["osquery", "checks", "detail", id, "results"] as const,
  results: (id: number | null, params?: QueryParams) =>
    [...checkKeys.resultsRoot(id), params ?? {}] as const,
};

export function checkQueryOptions(id: number | null) {
  return queryOptions<OsqueryCheck, ApiError>({
    queryKey: checkKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getOsqueryCheck({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function useChecks(params: CheckListParams = {}) {
  const queryParams = baseListParams(params);

  return useQuery<PageCheck, ApiError>({
    queryKey: checkKeys.list(queryParams),
    queryFn: ({ signal }) => unwrap(listOsqueryChecks({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: CHECK_REFRESH_MS,
  });
}

export function useCheck(id: number | null) {
  return useQuery(checkQueryOptions(id));
}

export function useCheckResults(id: number | null, params: CheckResultsParams = {}) {
  const queryParams = checkResultQueryParams(params);
  return useQuery<PageCheckHostStatus, ApiError>({
    queryKey: checkKeys.results(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listOsqueryCheckResults({
          path: detailPath(id),
          query: queryParams,
          signal,
        }),
      ),
    enabled: id !== null,
    refetchInterval: CHECK_REFRESH_MS,
  });
}

export function listAllCheckResults(
  id: number,
  params: CheckResultsParams = {},
): Promise<OsqueryCheckHostStatus[]> {
  return collectAllPages((page, perPage) =>
    unwrap(
      listOsqueryCheckResults({
        path: { id },
        query: checkResultQueryParams({ ...params, page, per_page: perPage }),
      }),
    ),
  );
}

function checkResultQueryParams(params: CheckResultsParams) {
  return {
    ...baseListParams(params),
    status: params.status,
  };
}

export function useCreateCheck() {
  const queryClient = useQueryClient();
  return useMutation<OsqueryCheck, ApiError, OsqueryCheckMutation>({
    mutationFn: (body) => unwrap(createOsqueryCheck({ body })),
    onSuccess: async () => {
      toast.add({ title: "Check created", type: "success" });
      await queryClient.invalidateQueries({ queryKey: checkKeys.all });
    },
  });
}

export function useUpdateCheck(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<OsqueryCheck, ApiError, OsqueryCheckMutation>({
    mutationFn: (body) =>
      unwrap(
        updateOsqueryCheck({
          path: detailPath(id),
          body,
        }),
      ),
    onSuccess: async (saved) => {
      const previous = queryClient.getQueryData<OsqueryCheck>(checkKeys.detail(id));
      const queryChanged = previous !== undefined && previous.query !== saved.query;
      if (queryChanged) {
        queryClient.removeQueries({ queryKey: checkKeys.resultsRoot(id) });
      }
      toast.add({
        title: queryChanged ? "Check saved and results cleared" : "Check saved",
        type: "success",
      });
      await queryClient.invalidateQueries({ queryKey: checkKeys.all });
    },
  });
}

export function useDeleteCheck() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteOsqueryCheck({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: checkKeys.all });
    },
  });
}

export function useBulkDeleteChecks() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteOsqueryChecks({ query: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: checkKeys.all });
    },
  });
}
