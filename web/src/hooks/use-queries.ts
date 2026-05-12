import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type SavedQuery = Schemas["Query"];
export type QueryListResult = Schemas["QueryListOutputBody"];
export type QueryMutation = Schemas["QueryMutationBody"];
export type QueryResults = Schemas["QueryResultsOutputBody"];

export interface QueryListParams {
  q?: string;
  platform?: string;
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: string;
}

export function useQueries(params: QueryListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    platform: nonEmpty(params.platform),
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: nonEmpty(params.order_key),
    order_direction: nonEmpty(params.order_direction),
  };

  return useQuery<QueryListResult, ApiError>({
    queryKey: queryKeys.queries(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/queries", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useQueryDetail(id: string) {
  return useQuery<SavedQuery, ApiError>({
    queryKey: queryKeys.query(id),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/queries/{id}", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useQueryResults(id: string) {
  return useQuery<QueryResults, ApiError>({
    queryKey: queryKeys.queryResults(id),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/queries/{id}/results", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useCreateQuery() {
  const queryClient = useQueryClient();
  return useMutation<SavedQuery, ApiError, QueryMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/queries", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["queries"] });
    },
  });
}

export function useUpdateQuery(id: string) {
  const queryClient = useQueryClient();
  return useMutation<SavedQuery, ApiError, QueryMutation>({
    mutationFn: (body) => unwrap(apiClient.PUT("/api/queries/{id}", { params: { path: { id } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.queries() });
      void queryClient.invalidateQueries({ queryKey: queryKeys.query(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.queryResults(id) });
    },
  });
}

export function useDeleteQuery() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/queries/{id}", { params: { path: { id: String(id) } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["queries"] });
    },
  });
}

export function useBulkDeleteQueries() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(apiClient.POST("/api/queries/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["queries"] });
    },
  });
}
