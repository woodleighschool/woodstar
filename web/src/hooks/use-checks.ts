import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import type { QueryablePlatform } from "@/lib/targeting";
import { nonEmpty } from "@/lib/utils";

export type Check = Schemas["Check"];
export type CheckListResult = Schemas["PaginatedBodyCheck"];
export type CheckMutation = Schemas["CheckCreate"];
export type CheckHosts = Schemas["ItemsBodyCheckHostStatus"];
export type CheckHostStatus = Schemas["CheckHostStatus"];

export interface CheckListParams {
  q?: string;
  platform?: QueryablePlatform;
  page_index?: number;
  page_size?: number;
  sort?: string;
}

export function useChecks(params: CheckListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    platform: params.platform,
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
  };

  return useQuery<CheckListResult, ApiError>({
    queryKey: queryKeys.checks(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/osquery/checks", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useCheck(id: string) {
  return useQuery<Check, ApiError>({
    queryKey: queryKeys.check(id),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/osquery/checks/{id}", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useCheckHosts(id: string) {
  return useQuery<CheckHosts, ApiError>({
    queryKey: queryKeys.checkHosts(id),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/osquery/checks/{id}/hosts", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useCreateCheck() {
  const queryClient = useQueryClient();
  return useMutation<Check, ApiError, CheckMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/osquery/checks", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["checks"] });
    },
  });
}

export function useUpdateCheck(id: string) {
  const queryClient = useQueryClient();
  return useMutation<Check, ApiError, CheckMutation>({
    mutationFn: (body) => unwrap(apiClient.PUT("/api/osquery/checks/{id}", { params: { path: { id } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.checks() });
      void queryClient.invalidateQueries({ queryKey: queryKeys.check(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.checkHosts(id) });
    },
  });
}

export function useDeleteCheck() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/osquery/checks/{id}", { params: { path: { id: String(id) } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["checks"] });
    },
  });
}

export function useBulkDeleteChecks() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(apiClient.POST("/api/osquery/checks/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["checks"] });
    },
  });
}
