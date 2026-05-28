import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import type { ListLabelsData } from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type Label = Schemas["Label"];
export type LabelListResult = Schemas["PaginatedBodyLabel"];
export type LabelCreate = Schemas["LabelCreateBody"];
export type LabelMutation = Schemas["LabelMutationBody"];
export type LabelListParams = NonNullable<ListLabelsData["query"]>;

export function useLabels(params: LabelListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
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
  return useMutation<Label, ApiError, LabelCreate>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/labels", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["labels"] });
      void queryClient.invalidateQueries({ queryKey: ["hosts"] });
    },
  });
}

export function useUpdateLabel(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<Label, ApiError, LabelMutation>({
    mutationFn: (body) => unwrap(apiClient.PUT("/api/labels/{id}", { params: { path: { id: id ?? 0 } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["labels"] });
      void queryClient.invalidateQueries({ queryKey: ["hosts"] });
    },
  });
}

export function useDeleteLabel() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/labels/{id}", { params: { path: { id } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["labels"] });
      void queryClient.invalidateQueries({ queryKey: ["hosts"] });
    },
  });
}
