import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError, Page, Rule, RuleMutation, RuleReferenceCandidate } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import type { ListSantaRuleReferencesData, ListSantaRulesData } from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type SantaRule = Rule;
export type SantaRuleMutation = RuleMutation;
export type SantaRuleListResult = Page<SantaRule>;
export type SantaRuleReference = RuleReferenceCandidate;
export type SantaRuleReferenceListResult = SantaRuleReference[];
export type SantaRuleType = SantaRule["rule_type"];
export type SantaRulePolicy = SantaRule["targets"]["include"][number]["policy"];

export type SantaRuleListParams = NonNullable<ListSantaRulesData["query"]>;
export type SantaRuleReferenceListParams = NonNullable<ListSantaRuleReferencesData["query"]>;

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
      unwrap(
        apiClient.GET("/api/santa/rules/{id}", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useSantaRuleReferences(params: SantaRuleReferenceListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    rule_type: params.rule_type,
    limit: params.limit ?? 20,
  };

  return useQuery<SantaRuleReferenceListResult, ApiError>({
    queryKey: queryKeys.santaRuleReferences(queryParams),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/rule-references", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useCreateSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<SantaRule, ApiError, SantaRuleMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/santa/rules", { body })),
    meta: { inlineError: true },
    onSuccess: (rule) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRule(rule.id) });
    },
  });
}

export function useUpdateSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<SantaRule, ApiError, { id: number; body: SantaRuleMutation }>({
    mutationFn: ({ id, body }) => unwrap(apiClient.PUT("/api/santa/rules/{id}", { params: { path: { id } }, body })),
    meta: { inlineError: true },
    onSuccess: (rule) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRule(rule.id) });
    },
  });
}

export function useDeleteSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/santa/rules/{id}", { params: { path: { id } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll });
    },
  });
}

export function useBulkDeleteSantaRules() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(apiClient.POST("/api/santa/rules/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll });
    },
  });
}
