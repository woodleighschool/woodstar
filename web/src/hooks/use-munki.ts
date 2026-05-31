import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  ApiError,
  MunkiArtifact,
  MunkiArtifactMutation,
  MunkiArtifactPage,
  MunkiAssignment,
  MunkiAssignmentMutation,
  MunkiAssignmentPage,
  MunkiRelease,
  MunkiReleaseMutation,
  MunkiReleasePage,
  MunkiSoftwareTitle,
  MunkiSoftwareTitleMutation,
  MunkiSoftwareTitlePage,
} from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type {
  MunkiArtifact,
  MunkiArtifactMutation,
  MunkiAssignment,
  MunkiAssignmentMutation,
  MunkiRelease,
  MunkiReleaseMutation,
  MunkiSoftwareTitle,
  MunkiSoftwareTitleMutation,
};
export type MunkiListResult<T> = {
  items: T[] | null;
  count: number;
};

export interface MunkiListParams {
  q?: string;
  page_index?: number;
  page_size?: number;
  sort?: string;
}

function queryParams(params: MunkiListParams) {
  return {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
  };
}

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

export function useMunkiArtifacts(params: MunkiListParams = {}) {
  const query = queryParams(params);
  return useQuery<MunkiArtifactPage, ApiError>({
    queryKey: queryKeys.munkiArtifacts(query),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/artifacts", {
          params: { query },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiReleases(params: MunkiListParams = {}) {
  const query = queryParams(params);
  return useQuery<MunkiReleasePage, ApiError>({
    queryKey: queryKeys.munkiReleases(query),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/releases", {
          params: { query },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiAssignments(params: MunkiListParams = {}) {
  const query = queryParams(params);
  return useQuery<MunkiAssignmentPage, ApiError>({
    queryKey: queryKeys.munkiAssignments(query),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/assignments", {
          params: { query },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useCreateMunkiSoftwareTitle() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareTitle, ApiError, MunkiSoftwareTitleMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/software-titles", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "software-titles"] });
    },
  });
}

export function useCreateMunkiArtifact() {
  const queryClient = useQueryClient();
  return useMutation<MunkiArtifact, ApiError, MunkiArtifactMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/artifacts", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "artifacts"] });
    },
  });
}

export function useCreateMunkiRelease() {
  const queryClient = useQueryClient();
  return useMutation<MunkiRelease, ApiError, MunkiReleaseMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/releases", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "releases"] });
    },
  });
}

export function useCreateMunkiAssignment() {
  const queryClient = useQueryClient();
  return useMutation<MunkiAssignment, ApiError, MunkiAssignmentMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/assignments", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "assignments"] });
    },
  });
}
