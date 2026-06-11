import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type { ApiError, Label, LabelMutation, Page } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import type { ListLabelsData } from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { Label, LabelMutation };
export type LabelListResult = Page<Label>;
export type LabelListParams = NonNullable<ListLabelsData["query"]>;

export function useLabels(params: LabelListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
    label_type: params.label_type,
    label_membership_type: params.label_membership_type,
  };

  return useQuery<LabelListResult, ApiError>({
    queryKey: queryKeys.labels(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/labels", {
          params: { query: queryParams },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useLabel(id: number | null) {
  return useQuery<Label, ApiError>({
    queryKey: queryKeys.label(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/labels/{id}", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateLabel() {
  const queryClient = useQueryClient();
  return useMutation<Label, ApiError, LabelMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/labels", { body })),
    meta: { inlineError: true },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.labelsAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

export function useUpdateLabel(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<Label, ApiError, LabelMutation>({
    mutationFn: (body) => unwrap(apiClient.PUT("/api/labels/{id}", { params: { path: { id: id ?? 0 } }, body })),
    meta: { inlineError: true },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.labelsAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

export function useDeleteLabel() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/labels/{id}", { params: { path: { id } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.labelsAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}
