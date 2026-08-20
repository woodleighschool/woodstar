import {
  keepPreviousData,
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import { hostKeys } from "@features/hosts/queries";
import type { ApiError, Label, LabelMutation, PageLabel } from "@lib/api";
import { createLabel, deleteLabel, getLabel, listLabels, unwrap, updateLabel } from "@lib/api";
import type { ListLabelsData } from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

type QueryParams = Record<string, unknown>;

const labelKeys = {
  all: ["labels"] as const,
  list: (params?: QueryParams) => ["labels", "list", params ?? {}] as const,
  detail: (id: number | null) => ["labels", "detail", id] as const,
};

export type LabelListParams = NonNullable<ListLabelsData["query"]>;

type RefetchOptions = { refetchInterval?: number | false };

export function labelQueryOptions(id: number | null) {
  return queryOptions<Label, ApiError>({
    queryKey: labelKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getLabel({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function useLabels(params: LabelListParams = {}, options: RefetchOptions = {}) {
  const queryParams = {
    ...baseListParams(params),
    label_type: params.label_type,
    label_membership_type: params.label_membership_type,
  };

  return useQuery<PageLabel, ApiError>({
    queryKey: labelKeys.list(queryParams),
    queryFn: ({ signal }) => unwrap(listLabels({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: options.refetchInterval,
  });
}

export function useLabel(id: number | null) {
  return useQuery(labelQueryOptions(id));
}

export function useCreateLabel() {
  const queryClient = useQueryClient();
  return useMutation<Label, ApiError, LabelMutation>({
    mutationFn: (body) => unwrap(createLabel({ body })),
    onSuccess: async () => {
      toast.add({ title: "Label Created", type: "success" });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: labelKeys.all }),
        queryClient.invalidateQueries({ queryKey: hostKeys.all }),
      ]);
    },
  });
}

export function useUpdateLabel(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<Label, ApiError, LabelMutation>({
    mutationFn: (body) => unwrap(updateLabel({ path: detailPath(id), body })),
    onSuccess: async () => {
      toast.add({ title: "Label Saved", type: "success" });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: labelKeys.all }),
        queryClient.invalidateQueries({ queryKey: hostKeys.all }),
      ]);
    },
  });
}

export function useDeleteLabel() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteLabel({ path: { id } })),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: labelKeys.all }),
        queryClient.invalidateQueries({ queryKey: hostKeys.all }),
      ]);
    },
  });
}
