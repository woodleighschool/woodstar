import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import type {
  ListSantaConfigurationsData,
  ListSantaEventsData,
  ListSantaFileAccessEventsData,
  ListSantaRulesData,
  ListSantaRuleTargetsData,
} from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type SantaConfiguration = Schemas["Configuration"];
export type SantaConfigurationMutation = Schemas["ConfigurationMutation"];
export type SantaConfigurationListResult = Schemas["PaginatedBodyConfiguration"];
export type SantaEvent = Schemas["ExecutionEvent"];
export type SantaEventListResult = Schemas["PaginatedBodyExecutionEvent"];
export type SantaFileAccessEvent = Schemas["FileAccessEvent"];
export type SantaFileAccessEventListResult = Schemas["PaginatedBodyFileAccessEvent"];
export type SantaHostSummary = Schemas["HostSummary"];
export type SantaRule = Schemas["Rule"];
export type SantaRuleMutation = Schemas["RuleMutation"];
export type SantaRuleListResult = Schemas["PaginatedBodyRule"];
export type SantaRuleTarget = Schemas["RuleTarget"];
export type SantaRuleTargetListResult = Schemas["ItemsBodyRuleTarget"];
export type SantaClientMode = Schemas["HostState"]["client_mode_reported"] | SantaConfiguration["client_mode"];
export type SantaExecutionDecision = SantaEvent["decision"];
export type SantaFileAccessDecision = SantaFileAccessEvent["decision"];
export type SantaRuleType = SantaRule["rule_type"];
export type SantaRulePolicy = NonNullable<SantaRule["includes"]>[number]["policy"];

export type SantaListParams = NonNullable<ListSantaConfigurationsData["query"]>;
export type SantaRuleListParams = NonNullable<ListSantaRulesData["query"]>;
export type SantaRuleTargetListParams = NonNullable<ListSantaRuleTargetsData["query"]>;
export type SantaEventListParams = NonNullable<ListSantaEventsData["query"]>;
export type SantaFileAccessEventListParams = NonNullable<ListSantaFileAccessEventsData["query"]>;
export type SantaEventDecisionFilter = NonNullable<NonNullable<SantaEventListParams["decisions"]>[number]>;

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

export function useSantaConfiguration(id: number | null) {
  return useQuery<SantaConfiguration, ApiError>({
    queryKey: queryKeys.santaConfiguration(id),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/configurations/{id}", { params: { path: { id: id ?? 0 } }, signal })),
    enabled: id !== null,
  });
}

export function useSantaRules(params: SantaRuleListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    rule_type: params.rule_type,
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

export function useSantaRule(id: number | null) {
  return useQuery<SantaRule, ApiError>({
    queryKey: queryKeys.santaRule(id),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/rules/{id}", { params: { path: { id: id ?? 0 } }, signal })),
    enabled: id !== null,
  });
}

export function useSantaRuleTargets(params: SantaRuleTargetListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    target_type: params.target_type,
    limit: params.limit ?? 20,
  };

  return useQuery<SantaRuleTargetListResult, ApiError>({
    queryKey: queryKeys.santaRuleTargets(queryParams),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/rule-targets", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaEvents(params: SantaEventListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    host_id: params.host_id,
    decisions: params.decisions && params.decisions.length > 0 ? params.decisions : undefined,
    since: nonEmpty(params.since),
    user: nonEmpty(params.user),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
  };

  return useQuery<SantaEventListResult, ApiError>({
    queryKey: queryKeys.santaEvents(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/santa/events", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaEvent(id: number | null) {
  return useQuery<SantaEvent, ApiError>({
    queryKey: queryKeys.santaEvent(id),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/events/{id}", { params: { path: { id: id ?? 0 } }, signal })),
    enabled: id !== null,
  });
}

export function useSantaFileAccessEvents(params: SantaFileAccessEventListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    host_id: params.host_id,
    decisions: params.decisions && params.decisions.length > 0 ? params.decisions : undefined,
    since: nonEmpty(params.since),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
  };

  return useQuery<SantaFileAccessEventListResult, ApiError>({
    queryKey: queryKeys.santaFileAccessEvents(queryParams),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/file-access-events", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaFileAccessEvent(id: number | null) {
  return useQuery<SantaFileAccessEvent, ApiError>({
    queryKey: queryKeys.santaFileAccessEvent(id),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/file-access-events/{id}", { params: { path: { id: id ?? 0 } }, signal })),
    enabled: id !== null,
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
      unwrap(apiClient.PATCH("/api/santa/configurations/{id}", { params: { path: { id } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "configurations"] });
    },
  });
}

export function useDeleteSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/santa/configurations/{id}", { params: { path: { id } } })),
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
    mutationFn: ({ id, body }) => unwrap(apiClient.PATCH("/api/santa/rules/{id}", { params: { path: { id } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "rules"] });
    },
  });
}

export function useDeleteSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/santa/rules/{id}", { params: { path: { id } } })),
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
          params: { path: { id: ruleID ?? 0 } },
          body: { ordered_include_ids },
        }),
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["santa", "rules"] });
    },
  });
}
