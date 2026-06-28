import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type {
  ApiError,
  OsqueryCheck,
  OsqueryCheckHostStatus,
  OsqueryCheckMutation,
  PageCheck,
} from "@/lib/api";
import {
  bulkDeleteOsqueryChecks,
  createOsqueryCheck,
  deleteOsqueryCheck,
  getOsqueryCheck,
  listOsqueryCheckResults,
  listOsqueryChecks,
  unwrap,
  updateOsqueryCheck,
} from "@/lib/api";
import type {
  ListOsqueryCheckResultsData,
  ListOsqueryChecksData,
} from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export type CheckListParams = NonNullable<ListOsqueryChecksData["query"]>;
export type CheckResultsParams = NonNullable<ListOsqueryCheckResultsData["query"]>;

export function useChecks(params: CheckListParams = {}) {
  const queryParams = baseListParams(params);

  return useQuery<PageCheck, ApiError>({
    queryKey: queryKeys.checks(queryParams),
    queryFn: ({ signal }) => unwrap(listOsqueryChecks({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useCheck(id: number | null) {
  return useQuery<OsqueryCheck, ApiError>({
    queryKey: queryKeys.check(id),
    queryFn: ({ signal }) =>
      unwrap(
        getOsqueryCheck({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCheckResults(id: number | null, params: CheckResultsParams = {}) {
  const queryParams = {
    response: params.response,
  };

  return useQuery<OsqueryCheckHostStatus[], ApiError>({
    queryKey: queryKeys.checkResults(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listOsqueryCheckResults({
          path: detailPath(id),
          query: queryParams,
          signal,
        }),
      ),
    enabled: id !== null,
    placeholderData: keepPreviousData,
  });
}

export function useCreateCheck() {
  const queryClient = useQueryClient();
  return useMutation<OsqueryCheck, ApiError, OsqueryCheckMutation>({
    mutationFn: (body) => unwrap(createOsqueryCheck({ body })),
    onSuccess: async () => {
      toast.success("Check created");
      await queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
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
    onSuccess: async () => {
      toast.success("Check saved");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.checksAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.check(id) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.checkResults(id) }),
      ]);
    },
  });
}

export function useDeleteCheck() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteOsqueryCheck({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
    },
  });
}

export function useBulkDeleteChecks() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteOsqueryChecks({ body: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
    },
  });
}
