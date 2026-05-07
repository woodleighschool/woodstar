import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { ApiError, apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type Label = Schemas["LabelBody"];
export type LabelListResult = Schemas["LabelListBody"];
export type LabelMutation = Schemas["LabelMutationBody"];

export interface LabelListParams {
  q?: string;
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: string;
  kind?: string;
  membership_type?: string;
  platform?: string;
}

export function useLabels(params: LabelListParams = {}) {
  const queryParams = {
    q: params.q?.trim() || undefined,
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: params.order_key || undefined,
    order_direction: params.order_direction || undefined,
    kind: params.kind || undefined,
    membership_type: params.membership_type || undefined,
    platform: params.platform || undefined,
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

export function useLabel(id: string) {
  return useQuery<Label, ApiError>({
    queryKey: queryKeys.label(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/labels/{id}", {
          params: { path: { id } },
          signal,
        }),
      ),
    enabled: id !== "",
  });
}

export function useCreateLabel() {
  const queryClient = useQueryClient();
  return useMutation<Label, ApiError, LabelMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/labels", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["labels"] });
      void queryClient.invalidateQueries({ queryKey: ["hosts"] });
    },
  });
}

export function useUpdateLabel(id: string) {
  const queryClient = useQueryClient();
  return useMutation<Label, ApiError, LabelMutation>({
    mutationFn: (body) => unwrap(apiClient.PATCH("/api/labels/{id}", { params: { path: { id } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["labels"] });
      void queryClient.invalidateQueries({ queryKey: ["hosts"] });
    },
  });
}

export function useDeleteLabel() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, string>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/labels/{id}", { params: { path: { id } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["labels"] });
      void queryClient.invalidateQueries({ queryKey: ["hosts"] });
    },
  });
}
