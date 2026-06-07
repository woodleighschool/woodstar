import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  ApiError,
  MunkiPackage,
  MunkiPackageImportMutation,
  MunkiPackageMutation,
  MunkiPackagePage,
} from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

import { softwareQueryParams, type MunkiSoftwareListParams } from "./shared";

export type { MunkiPackage, MunkiPackageImportMutation, MunkiPackageMutation };

export function useMunkiPackages(params: MunkiSoftwareListParams = {}) {
  const query = softwareQueryParams(params);
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
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<MunkiPackage, ApiError, { softwareID: number; body: MunkiPackageMutation }>({
    mutationFn: ({ softwareID, body }) =>
      unwrap(apiClient.POST("/api/munki/software/{id}/packages", { params: { path: { id: softwareID } }, body })),
    onSuccess: (pkg) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "packages"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(pkg.software_id) });
    },
  });
}

export function useUpdateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<MunkiPackage, ApiError, { id: number; body: MunkiPackageMutation }>({
    mutationFn: ({ id, body }) => unwrap(apiClient.PUT("/api/munki/packages/{id}", { params: { path: { id } }, body })),
    onSuccess: (pkg) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "packages"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackage(pkg.id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(pkg.software_id) });
    },
  });
}

export function useImportMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<MunkiPackage, ApiError, { softwareID: number; body: MunkiPackageImportMutation }>({
    mutationFn: ({ softwareID, body }) =>
      unwrap(
        apiClient.POST("/api/munki/software/{id}/packages/import", { params: { path: { id: softwareID } }, body }),
      ),
    onSuccess: (pkg) => {
      void queryClient.invalidateQueries({ queryKey: ["munki", "packages"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(pkg.software_id) });
    },
  });
}
