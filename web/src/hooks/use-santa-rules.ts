import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type {
  ApiError,
  PageRule,
  SantaRule,
  SantaRuleMutation,
  SantaRuleReferenceCandidate,
} from "@/lib/api";
import {
  bulkDeleteSantaRules,
  createSantaRule,
  deleteSantaRule,
  getSantaRule,
  listSantaRuleReferences,
  listSantaRules,
  unwrap,
  updateSantaRule,
} from "@/lib/api";
import type { ListSantaRuleReferencesData, ListSantaRulesData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { SantaRule, SantaRuleMutation };
export type SantaRuleListResult = PageRule;
export type SantaRuleReference = SantaRuleReferenceCandidate;
export type SantaRuleReferenceListResult = SantaRuleReference[];
export type SantaRuleType = SantaRule["rule_type"];
export type SantaRulePolicy = SantaRule["targets"]["include"][number]["policy"];

export type SantaRuleListParams = NonNullable<ListSantaRulesData["query"]>;
export type SantaRuleReferenceListParams = NonNullable<ListSantaRuleReferencesData["query"]>;

export function useSantaRules(params: SantaRuleListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    rule_type: params.rule_type,
  };

  return useQuery<SantaRuleListResult, ApiError>({
    queryKey: queryKeys.santaRules(queryParams),
    queryFn: ({ signal }) => unwrap(listSantaRules({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaRule(id: number | null) {
  return useQuery<SantaRule, ApiError>({
    queryKey: queryKeys.santaRule(id),
    queryFn: ({ signal }) =>
      unwrap(
        getSantaRule({
          path: { id: id ?? 0 },
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
    queryFn: ({ signal }) => unwrap(listSantaRuleReferences({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useCreateSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<SantaRule, ApiError, SantaRuleMutation>({
    mutationFn: (body) => unwrap(createSantaRule({ body })),
    onSuccess: (rule) => {
      toast.success("Rule created");
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRule(rule.id) });
    },
  });
}

export function useUpdateSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<SantaRule, ApiError, { id: number; body: SantaRuleMutation }>({
    mutationFn: ({ id, body }) => unwrap(updateSantaRule({ path: { id }, body })),
    onSuccess: (rule) => {
      toast.success("Rule saved");
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRule(rule.id) });
    },
  });
}

export function useDeleteSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteSantaRule({ path: { id } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll });
    },
  });
}

export function useBulkDeleteSantaRules() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteSantaRules({ body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll });
    },
  });
}
