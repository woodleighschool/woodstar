import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type { ApiError, Check, CheckHostStatus, CheckMutation, Page } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { Check, CheckHostStatus, CheckMutation };
export type CheckListResult = Page<Check>;
export type CheckHosts = CheckHostStatus[];

export interface CheckListParams {
  q?: string;
  page?: number;
  per_page?: number;
  sort?: string;
}

export function useChecks(params: CheckListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };

  return useQuery<CheckListResult, ApiError>({
    queryKey: queryKeys.checks(queryParams),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/osquery/checks", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useCheck(id: number | null) {
  return useQuery<Check, ApiError>({
    queryKey: queryKeys.check(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/osquery/checks/{id}", {
          params: { path: { id: id ?? 0 } },
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
        apiClient.GET("/api/osquery/checks/{id}/hosts", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateCheck() {
  const queryClient = useQueryClient();
  return useMutation<Check, ApiError, CheckMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/osquery/checks", { body })),
    meta: { inlineError: true },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
    },
  });
}

export function useUpdateCheck(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<Check, ApiError, CheckMutation>({
    mutationFn: (body) =>
      unwrap(
        apiClient.PUT("/api/osquery/checks/{id}", {
          params: { path: { id: id ?? 0 } },
          body,
        }),
      ),
    meta: { inlineError: true },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.check(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.checkHosts(id) });
    },
  });
}

export function useDeleteCheck() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) =>
      unwrap(apiClient.DELETE("/api/osquery/checks/{id}", { params: { path: { id } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
    },
  });
}

export function useBulkDeleteChecks() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) =>
      unwrap(apiClient.POST("/api/osquery/checks/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.checksAll });
    },
  });
}
