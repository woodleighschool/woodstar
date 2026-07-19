import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type { ApiError, Label, LabelMutation, PageLabel } from "@/lib/api";
import { createLabel, deleteLabel, listLabels, unwrap, updateLabel } from "@/lib/api";
import type { ListLabelsData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { labelQueryOptions } from "@/lib/queries/labels";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export type LabelListParams = NonNullable<ListLabelsData["query"]>;

type RefetchOptions = { refetchInterval?: number | false };

export function useLabels(params: LabelListParams = {}, options: RefetchOptions = {}) {
  const queryParams = {
    ...baseListParams(params),
    label_type: params.label_type,
    label_membership_type: params.label_membership_type,
  };

  return useQuery<PageLabel, ApiError>({
    queryKey: queryKeys.labels(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listLabels({
          query: queryParams,
          signal,
        }),
      ),
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
      toast.success("Label created");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.labelsAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll }),
      ]);
    },
  });
}

export function useUpdateLabel(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<Label, ApiError, LabelMutation>({
    mutationFn: (body) => unwrap(updateLabel({ path: detailPath(id), body })),
    onSuccess: async () => {
      toast.success("Label saved");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.labelsAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll }),
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
        queryClient.invalidateQueries({ queryKey: queryKeys.labelsAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll }),
      ]);
    },
  });
}
