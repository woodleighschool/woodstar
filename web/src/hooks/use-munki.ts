import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  ApiError,
  MunkiArtifact,
  MunkiArtifactMutation,
  MunkiArtifactUpload,
  MunkiArtifactUploadMutation,
  MunkiAssignment,
  MunkiAssignmentMutation,
  MunkiAssignmentPage,
  MunkiPackage,
  MunkiPackageImportMutation,
  MunkiPackageMutation,
  MunkiPackagePage,
  MunkiSoftwareTitle,
  MunkiSoftwareTitleDetail,
  MunkiSoftwareTitleMutation,
  MunkiSoftwareTitlePage,
} from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type {
  MunkiArtifact,
  MunkiArtifactMutation,
  MunkiArtifactUpload,
  MunkiArtifactUploadMutation,
  MunkiAssignment,
  MunkiAssignmentMutation,
  MunkiPackage,
  MunkiPackageImportMutation,
  MunkiPackageMutation,
  MunkiSoftwareTitle,
  MunkiSoftwareTitleDetail,
  MunkiSoftwareTitleMutation,
};

export interface MunkiListParams {
  q?: string;
  page_index?: number;
  page_size?: number;
  sort?: string;
}

export interface MunkiScopedListParams extends MunkiListParams {
  software_id?: number;
}

function queryParams(params: MunkiListParams) {
  return {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
  };
}

function scopedQueryParams(params: MunkiScopedListParams) {
  return {
    ...queryParams(params),
    software_id: params.software_id === 0 ? undefined : params.software_id,
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

export function useMunkiPackages(params: MunkiScopedListParams = {}) {
  const query = scopedQueryParams(params);
  return useQuery<MunkiPackagePage, ApiError>({
    queryKey: queryKeys.munkiPackages(query),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/packages", {
          params: { query },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiPackage(id: number | null) {
  return useQuery<MunkiPackage, ApiError>({
    queryKey: queryKeys.munkiPackage(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/packages/{id}", {
          params: { path: { id } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useMunkiAssignments(params: MunkiScopedListParams = {}) {
  const query = scopedQueryParams(params);
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

export function useMunkiAssignment(id: number | null) {
  return useQuery<MunkiAssignment, ApiError>({
    queryKey: queryKeys.munkiAssignment(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/assignments/{id}", {
          params: { path: { id } },
          signal,
        }),
      ),
    enabled: id !== null,
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

export function useUpdateMunkiSoftwareTitle() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareTitle, ApiError, { id: number; body: MunkiSoftwareTitleMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(apiClient.PATCH("/api/munki/software-titles/{id}", { params: { path: { id } }, body })),
    onSuccess: (title) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "software-titles"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareTitle(title.id) });
    },
  });
}

export function useCreateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<MunkiPackage, ApiError, MunkiPackageMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/packages", { body })),
    onSuccess: (pkg) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "packages"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareTitle(pkg.software_id) });
    },
  });
}

export function useUpdateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<MunkiPackage, ApiError, { id: number; body: MunkiPackageMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(apiClient.PATCH("/api/munki/packages/{id}", { params: { path: { id } }, body })),
    onSuccess: (pkg) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "packages"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackage(pkg.id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareTitle(pkg.software_id) });
    },
  });
}

export function useImportMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<MunkiPackage, ApiError, MunkiPackageImportMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/packages/import", { body })),
    onSuccess: (pkg) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "packages"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareTitle(pkg.software_id) });
    },
  });
}

export function useCreateMunkiArtifactUpload() {
  return useMutation<MunkiArtifactUpload, ApiError, MunkiArtifactUploadMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/artifact-uploads", { body })),
  });
}

export function useCreateMunkiArtifact() {
  return useMutation<MunkiArtifact, ApiError, MunkiArtifactMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/artifacts", { body })),
  });
}

export function useCreateMunkiAssignment() {
  const queryClient = useQueryClient();
  return useMutation<MunkiAssignment, ApiError, MunkiAssignmentMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/assignments", { body })),
    onSuccess: (assignment) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "assignments"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareTitle(assignment.software_id) });
    },
  });
}

export function useUpdateMunkiAssignment() {
  const queryClient = useQueryClient();
  return useMutation<MunkiAssignment, ApiError, { id: number; body: MunkiAssignmentMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(apiClient.PATCH("/api/munki/assignments/{id}", { params: { path: { id } }, body })),
    onSuccess: (assignment) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "assignments"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiAssignment(assignment.id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareTitle(assignment.software_id) });
    },
  });
}

export function useReorderMunkiAssignments() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, { softwareId: number; orderedIds: number[] }>({
    mutationFn: ({ softwareId, orderedIds }) =>
      unwrap(
        apiClient.PUT("/api/munki/software-titles/{id}/assignments/order", {
          params: { path: { id: softwareId } },
          body: { ordered_ids: orderedIds },
        }),
      ),
    onSuccess: (_result, variables) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "assignments"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareTitle(variables.softwareId) });
    },
  });
}
