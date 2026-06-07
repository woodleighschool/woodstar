import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  ApiError,
  MunkiSoftwareTitle,
  MunkiSoftwareTitleDetail,
  MunkiSoftwareTitleMutation,
  MunkiSoftwareTitlePage,
} from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

import { queryParams, type MunkiListParams } from "./shared";

export type { MunkiSoftwareTitle, MunkiSoftwareTitleDetail, MunkiSoftwareTitleMutation };

export function useMunkiSoftwareTitles(params: MunkiListParams = {}) {
  const query = queryParams(params);
  return useQuery<MunkiSoftwareTitlePage, ApiError>({
    queryKey: queryKeys.munkiSoftwareTitles(query),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/software-titles", {
          params: { query },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiSoftwareTitle(id: number | null) {
  return useQuery<MunkiSoftwareTitleDetail, ApiError>({
    queryKey: queryKeys.munkiSoftwareTitle(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/software-titles/{id}", {
          params: { path: { id } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateMunkiSoftwareTitle() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareTitleDetail, ApiError, MunkiSoftwareTitleMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/software-titles", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "software-titles"] });
    },
  });
}

export function useUpdateMunkiSoftwareTitle() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareTitleDetail, ApiError, { id: number; body: MunkiSoftwareTitleMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(apiClient.PATCH("/api/munki/software-titles/{id}", { params: { path: { id } }, body })),
    onSuccess: (title) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "software-titles"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareTitle(title.id) });
    },
  });
}

export function useDeleteMunkiSoftwareTitle() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/munki/software-titles/{id}", { params: { path: { id } } })),
    onSuccess: (_data, id) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "software-titles"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareTitle(id) });
    },
  });
}

export function useBulkDeleteMunkiSoftwareTitles() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(apiClient.POST("/api/munki/software-titles/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "software-titles"] });
    },
  });
}
