import {
  keepPreviousData,
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";

import type { ApiError, PageRule, SantaRule, SantaRuleMutation } from "@lib/api";
import {
  bulkDeleteSantaRules,
  createSantaRule,
  getSantaRule,
  listSantaRules,
  unwrap,
  updateSantaRule,
} from "@lib/api";
import type { ListSantaRulesData } from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

type QueryParams = Record<string, unknown>;

type SantaRuleListParams = NonNullable<ListSantaRulesData["query"]>;

const ruleKeys = {
  all: ["santa", "rules"] as const,
  list: (params?: QueryParams) => ["santa", "rules", "list", params ?? {}] as const,
  detail: (id: number | null) => ["santa", "rules", "detail", id] as const,
};

export function santaRuleQueryOptions(id: number | null) {
  return queryOptions<SantaRule, ApiError>({
    queryKey: ruleKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getSantaRule({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function useSantaRules(params: SantaRuleListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    rule_type: params.rule_type,
  };

  return useQuery<PageRule, ApiError>({
    queryKey: ruleKeys.list(queryParams),
    queryFn: ({ signal }) => unwrap(listSantaRules({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaRule(id: number | null) {
  return useQuery(santaRuleQueryOptions(id));
}

export function useCreateSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<SantaRule, ApiError, SantaRuleMutation>({
    mutationFn: (body) => unwrap(createSantaRule({ body })),
    onSuccess: async () => {
      toast.success("Rule created");
      await queryClient.invalidateQueries({ queryKey: ruleKeys.all });
    },
  });
}

export function useUpdateSantaRule() {
  const queryClient = useQueryClient();
  return useMutation<SantaRule, ApiError, { id: number; body: SantaRuleMutation }>({
    mutationFn: ({ id, body }) => unwrap(updateSantaRule({ path: { id }, body })),
    onSuccess: async () => {
      toast.success("Rule saved");
      await queryClient.invalidateQueries({ queryKey: ruleKeys.all });
    },
  });
}

export function useBulkDeleteSantaRules() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteSantaRules({ query: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ruleKeys.all });
    },
  });
}
