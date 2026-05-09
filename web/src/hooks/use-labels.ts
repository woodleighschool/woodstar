import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type Label = Schemas["LabelBody"];
export type LabelListResult = Schemas["LabelListBody"];
export type LabelCreate = Schemas["LabelCreateBody"];
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
    q: nonEmpty(params.q),
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: nonEmpty(params.order_key),
    order_direction: nonEmpty(params.order_direction),
    kind: nonEmpty(params.kind),
    membership_type: nonEmpty(params.membership_type),
    platform: nonEmpty(params.platform),
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
    mutationFn: (body) => unwrap(apiClient.PUT("/api/labels/{id}", { params: { path: { id: String(id) } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["labels"] });
      void queryClient.invalidateQueries({ queryKey: ["hosts"] });
    },
  });
}

export function useDeleteLabel() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/labels/{id}", { params: { path: { id: String(id) } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["labels"] });
      void queryClient.invalidateQueries({ queryKey: ["hosts"] });
    },
  });
}
