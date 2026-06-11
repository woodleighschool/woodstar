import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type {
  ApiError,
  MunkiPackage,
  MunkiPackageCreateMutation,
  MunkiPackageMutation,
  MunkiPackagePage,
} from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { MunkiPackage, MunkiPackageCreateMutation, MunkiPackageMutation };

interface MunkiSoftwareListParams {
  q?: string;
  page?: number;
  per_page?: number;
  sort?: string;
  software_id?: number;
}

function softwareQueryParams(params: MunkiSoftwareListParams) {
  return {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
    software_id: params.software_id === 0 ? undefined : params.software_id,
  };
}

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
  return useMutation<MunkiPackage, ApiError, MunkiPackageCreateMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/packages", { body })),
    meta: { inlineError: true },
    onSuccess: (pkg) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(pkg.software_id) });
    },
  });
}

export function useUpdateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<MunkiPackage, ApiError, { id: number; body: MunkiPackageMutation }>({
    mutationFn: ({ id, body }) => unwrap(apiClient.PUT("/api/munki/packages/{id}", { params: { path: { id } }, body })),
    meta: { inlineError: true },
    onSuccess: (pkg) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackage(pkg.id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(pkg.software_id) });
    },
  });
}
