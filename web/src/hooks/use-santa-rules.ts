import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type { ApiError, PageRule, SantaRule, SantaRuleMutation } from "@/lib/api";
import {
  bulkDeleteSantaRules,
  createSantaRule,
  deleteSantaRule,
  getSantaRule,
  listSantaRules,
  unwrap,
  updateSantaRule,
} from "@/lib/api";
import type { ListSantaRulesData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export type SantaRuleListParams = NonNullable<ListSantaRulesData["query"]>;

export function useSantaRules(params: SantaRuleListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    rule_type: params.rule_type,
  };

  return useQuery<PageRule, ApiError>({
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
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<SantaRule, ApiError, SantaRuleMutation>({
    mutationFn: (body) => unwrap(createSantaRule({ body })),
    onSuccess: async (rule) => {
      toast.success("Rule created");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.santaRule(rule.id) }),
      ]);
    },
  });
}

export function useUpdateSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<SantaRule, ApiError, { id: number; body: SantaRuleMutation }>({
    mutationFn: ({ id, body }) => unwrap(updateSantaRule({ path: { id }, body })),
    onSuccess: async (rule) => {
      toast.success("Rule saved");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.santaRule(rule.id) }),
      ]);
    },
  });
}

export function useDeleteSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteSantaRule({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll });
    },
  });
}

export function useBulkDeleteSantaRules() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteSantaRules({ query: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.santaRulesAll });
    },
  });
}
