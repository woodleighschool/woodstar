import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type Check = Schemas["CheckBody"];
export type CheckListResult = Schemas["CheckListOutputBody"];
export type CheckMutation = Schemas["CheckMutationBody"];
export type CheckPut = Schemas["CheckPutBody"];
export type CheckHosts = Schemas["CheckHostsOutputBody"];
export type CheckHostStatus = Schemas["CheckHostBody"];

export interface CheckListParams {
  q?: string;
  platform?: string;
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: string;
}

export function useChecks(params: CheckListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    platform: nonEmpty(params.platform),
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: nonEmpty(params.order_key),
    order_direction: nonEmpty(params.order_direction),
  };

  return useQuery<CheckListResult, ApiError>({
    queryKey: queryKeys.checks(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/checks", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useCheck(id: string) {
  return useQuery<Check, ApiError>({
    queryKey: queryKeys.check(id),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/checks/{id}", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useCheckHosts(id: string) {
  return useQuery<CheckHosts, ApiError>({
    queryKey: queryKeys.checkHosts(id),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/checks/{id}/hosts", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useCreateCheck() {
  const queryClient = useQueryClient();
  return useMutation<Check, ApiError, CheckMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/checks", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["checks"] });
    },
  });
}

export function useUpdateCheck(id: string) {
  const queryClient = useQueryClient();
  return useMutation<Check, ApiError, CheckPut>({
    mutationFn: (body) => unwrap(apiClient.PUT("/api/checks/{id}", { params: { path: { id } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.checks() });
      void queryClient.invalidateQueries({ queryKey: queryKeys.check(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.checkHosts(id) });
    },
  });
}

export function useDeleteCheck() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, string>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/checks/{id}", { params: { path: { id } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["checks"] });
    },
  });
}
