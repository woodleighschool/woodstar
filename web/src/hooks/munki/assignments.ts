import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  ApiError,
  MunkiAssignment,
  MunkiAssignmentExcludesBody,
  MunkiAssignmentMutation,
  MunkiAssignmentPage,
} from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

import { scopedQueryParams, type MunkiScopedListParams } from "./shared";

export type { MunkiAssignment, MunkiAssignmentMutation };

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
        apiClient.PUT("/api/munki/software-titles/{id}/includes/order", {
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

export function useUpdateMunkiAssignmentExcludeLabels() {
  const queryClient = useQueryClient();
  return useMutation<MunkiAssignmentExcludesBody, ApiError, { softwareId: number; excludeLabelIds: number[] }>({
    mutationFn: ({ softwareId, excludeLabelIds }) =>
      unwrap(
        apiClient.PUT("/api/munki/software-titles/{id}/exclude-labels", {
          params: { path: { id: softwareId } },
          body: { exclude_label_ids: excludeLabelIds },
        }),
      ),
    onSuccess: (_result, variables) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareTitle(variables.softwareId) });
    },
  });
}
