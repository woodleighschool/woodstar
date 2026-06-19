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
  listOsqueryCheckHosts,
  listOsqueryChecks,
  unwrap,
  updateOsqueryCheck,
} from "@/lib/api";
import type { ListOsqueryChecksData } from "@/lib/api-client/types.gen";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { OsqueryCheck, OsqueryCheckHostStatus, OsqueryCheckMutation };
export type CheckListResult = PageCheck;
export type CheckHosts = OsqueryCheckHostStatus[];
export type CheckListParams = NonNullable<ListOsqueryChecksData["query"]>;

export function useChecks(params: CheckListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };

  return useQuery<CheckListResult, ApiError>({
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
          path: { id: id ?? 0 },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCheckHosts(id: number | null) {
  return useQuery<CheckHosts, ApiError>({
    queryKey: queryKeys.checkHosts(id),
    queryFn: ({ signal }) =>
      unwrap(
        listOsqueryCheckHosts({
          path: { id: id ?? 0 },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateCheck() {
  const queryClient = useQueryClient();
  return useMutation<OsqueryCheck, ApiError, OsqueryCheckMutation>({
    mutationFn: (body) => unwrap(createOsqueryCheck({ body })),
    onSuccess: () => {
      toast.success("Check created");
      void queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
    },
  });
}

export function useUpdateCheck(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<OsqueryCheck, ApiError, OsqueryCheckMutation>({
    mutationFn: (body) =>
      unwrap(
        updateOsqueryCheck({
          path: { id: id ?? 0 },
          body,
        }),
      ),
    onSuccess: () => {
      toast.success("Check saved");
      void queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.check(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.checkHosts(id) });
    },
  });
}

export function useDeleteCheck() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteOsqueryCheck({ path: { id } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
    },
  });
}

export function useBulkDeleteChecks() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteOsqueryChecks({ body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
    },
  });
}
