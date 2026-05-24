import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import type { paths } from "@/lib/api-schema";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type SantaConfiguration = Schemas["Configuration"];
export type SantaConfigurationMutation = Schemas["ConfigurationMutation"];
export type SantaConfigurationListResult = Schemas["PaginatedBodyConfiguration"];
export type SantaEvent = Schemas["ExecutionEvent"];
export type SantaEventPage = Schemas["EventPage"];
export type SantaRule = Schemas["Rule"];
export type SantaRuleMutation = Schemas["RuleMutation"];
export type SantaRuleListResult = Schemas["PaginatedBodyRule"];
export type SantaSyncToken = Schemas["SyncToken"];

export type SantaListParams = NonNullable<paths["/api/santa/configurations"]["get"]["parameters"]["query"]>;
export type SantaRuleListParams = NonNullable<paths["/api/santa/rules"]["get"]["parameters"]["query"]>;
export type SantaEventListParams = NonNullable<paths["/api/santa/events"]["get"]["parameters"]["query"]>;

export function useSantaConfigurations(params: SantaListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
  };

  return useQuery<SantaConfigurationListResult, ApiError>({
    queryKey: queryKeys.santaConfigurations(queryParams),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/configurations", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaConfiguration(id: string) {
  return useQuery<SantaConfiguration, ApiError>({
    queryKey: queryKeys.santaConfiguration(id),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/configurations/{id}", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useSantaRules(params: SantaRuleListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    rule_type: nonEmpty(params.rule_type),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
  };

  return useQuery<SantaRuleListResult, ApiError>({
    queryKey: queryKeys.santaRules(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/santa/rules", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaRule(id: string) {
  return useQuery<SantaRule, ApiError>({
    queryKey: queryKeys.santaRule(id),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/santa/rules/{id}", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useSantaEvents(params: SantaEventListParams = {}) {
  const queryParams = {
    host_id: params.host_id,
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

export function useCreateSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<SantaConfiguration, ApiError, SantaConfigurationMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/santa/configurations", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "configurations"] });
    },
  });
}

export function useUpdateSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<SantaConfiguration, ApiError, { id: number; body: SantaConfigurationMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(apiClient.PATCH("/api/santa/configurations/{id}", { params: { path: { id: String(id) } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "configurations"] });
    },
  });
}

export function useDeleteSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) =>
      unwrap(apiClient.DELETE("/api/santa/configurations/{id}", { params: { path: { id: String(id) } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "configurations"] });
    },
  });
}

export function useBulkDeleteSantaConfigurations() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(apiClient.POST("/api/santa/configurations/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "configurations"] });
    },
  });
}

export function useReorderSantaConfigurations() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ordered_ids) => unwrap(apiClient.PUT("/api/santa/configurations/order", { body: { ordered_ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "configurations"] });
    },
  });
}

export function useCreateSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<SantaRule, ApiError, SantaRuleMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/santa/rules", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "rules"] });
    },
  });
}

export function useUpdateSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<SantaRule, ApiError, { id: number; body: SantaRuleMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(apiClient.PATCH("/api/santa/rules/{id}", { params: { path: { id: String(id) } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "rules"] });
    },
  });
}

export function useDeleteSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/santa/rules/{id}", { params: { path: { id: String(id) } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "rules"] });
    },
  });
}

export function useBulkDeleteSantaRules() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(apiClient.POST("/api/santa/rules/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "rules"] });
    },
  });
}

export function useReorderSantaRuleIncludes(ruleID: number | null) {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ordered_include_ids) =>
      unwrap(
        apiClient.PUT("/api/santa/rules/{id}/includes/order", {
          params: { path: { id: String(ruleID) } },
          body: { ordered_include_ids },
        }),
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "rules"] });
    },
  });
}

export function useCreateSantaSyncToken() {
  const queryClient = useQueryClient();
  return useMutation<SantaSyncToken, ApiError>({
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
