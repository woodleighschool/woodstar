import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type { ApiError, MunkiSoftware, MunkiSoftwareDetail, MunkiSoftwareMutation, MunkiSoftwarePage } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { MunkiSoftware, MunkiSoftwareDetail, MunkiSoftwareMutation };

interface MunkiListParams {
  q?: string;
  page?: number;
  per_page?: number;
  sort?: string;
}

function queryParams(params: MunkiListParams) {
  return {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };
}

export function useMunkiSoftware(params: MunkiListParams = {}) {
  const query = queryParams(params);
  return useQuery<MunkiSoftwarePage, ApiError>({
    queryKey: queryKeys.munkiSoftware(query),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/software", {
          params: { query },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiSoftwareDetail(id: number | null) {
  return useQuery<MunkiSoftwareDetail, ApiError>({
    queryKey: queryKeys.munkiSoftwareDetail(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/software/{id}", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareDetail, ApiError, MunkiSoftwareMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/software", { body })),
    meta: { inlineError: true },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
    },
  });
}

export function useUpdateMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareDetail, ApiError, { id: number; body: MunkiSoftwareMutation }>({
    mutationFn: ({ id, body }) => unwrap(apiClient.PUT("/api/munki/software/{id}", { params: { path: { id } }, body })),
    meta: { inlineError: true },
    onSuccess: (title) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(title.id) });
    },
  });
}

export function useDeleteMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/munki/software/{id}", { params: { path: { id } } })),
    onSuccess: (_data, id) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(id) });
    },
  });
}

export function useBulkDeleteMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(apiClient.POST("/api/munki/software/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
    },
  });
}
