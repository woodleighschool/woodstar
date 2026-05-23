import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type SantaConfiguration = Schemas["Configuration"];
export type SantaConfigurationListResult = Schemas["PaginatedBodyConfiguration"];
export type SantaEvent = Schemas["ExecutionEvent"];
export type SantaEventPage = Schemas["EventPage"];
export type SantaRule = Schemas["Rule"];
export type SantaRuleListResult = Schemas["PaginatedBodyRule"];
export type SantaSyncToken = Schemas["SyncToken"];
export type CreatedSantaSyncToken = Schemas["CreatedSyncToken"];

export interface SantaListParams {
  q?: string;
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: string;
}

export interface SantaRuleListParams extends SantaListParams {
  rule_type?: string;
}

export interface SantaEventListParams {
  host_id?: string;
  decision?: string;
  since?: string;
  limit?: number;
  after?: string;
}

export function useSantaConfigurations(params: SantaListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: nonEmpty(params.order_key),
    order_direction: nonEmpty(params.order_direction),
  };

  return useQuery<SantaConfigurationListResult, ApiError>({
    queryKey: queryKeys.santaConfigurations(queryParams),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/configurations", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaRules(params: SantaRuleListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    rule_type: nonEmpty(params.rule_type),
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: nonEmpty(params.order_key),
    order_direction: nonEmpty(params.order_direction),
  };

  return useQuery<SantaRuleListResult, ApiError>({
    queryKey: queryKeys.santaRules(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/santa/rules", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaEvents(params: SantaEventListParams = {}) {
  const queryParams = {
    host_id: nonEmpty(params.host_id),
    decision: nonEmpty(params.decision),
    since: nonEmpty(params.since),
    limit: params.limit ?? 50,
    after: nonEmpty(params.after),
  };

  return useQuery<SantaEventPage, ApiError>({
    queryKey: queryKeys.santaEvents(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/santa/events", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaSyncTokens() {
  return useQuery<SantaSyncToken[], ApiError>({
    queryKey: queryKeys.santaSyncTokens,
    queryFn: async ({ signal }) => (await unwrap(apiClient.GET("/api/santa/sync-tokens", { signal }))) ?? [],
  });
}

export function useCreateSantaSyncToken() {
  const queryClient = useQueryClient();
  return useMutation<CreatedSantaSyncToken, ApiError>({
    mutationFn: () => unwrap(apiClient.POST("/api/santa/sync-tokens")),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaSyncTokens });
    },
  });
}

export function useDeleteSantaSyncToken() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) =>
      unwrap(apiClient.DELETE("/api/santa/sync-tokens/{id}", { params: { path: { id: String(id) } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaSyncTokens });
    },
  });
}
