import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import type {
  ApiError,
  MunkiPackage,
  MunkiPackageCreateMutation,
  MunkiPackageMutation,
  PageMunkiPackage,
} from "@/lib/api";
import {
  bulkDeleteMunkiPackages,
  createMunkiPackage,
  getMunkiPackage,
  listMunkiPackages,
  unwrap,
  updateMunkiPackage,
} from "@/lib/api";
import type { ListMunkiPackagesData } from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { MunkiPackage, MunkiPackageCreateMutation, MunkiPackageMutation };

type MunkiSoftwareListParams = NonNullable<ListMunkiPackagesData["query"]>;

function softwareQueryParams(params: MunkiSoftwareListParams) {
  return {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
    type: params.type?.length ? params.type : undefined,
    software_id: params.software_id === 0 ? undefined : params.software_id,
  };
}

export function useMunkiPackages(params: MunkiSoftwareListParams = {}) {
  const query = softwareQueryParams(params);
  return useQuery<PageMunkiPackage, ApiError>({
    queryKey: queryKeys.munkiPackages(query),
    queryFn: ({ signal }) => unwrap(listMunkiPackages({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiPackage(id: number | null) {
  return useQuery<MunkiPackage, ApiError>({
    queryKey: queryKeys.munkiPackage(id),
    queryFn: ({ signal }) => unwrap(getMunkiPackage({ path: { id: id ?? 0 }, signal })),
    enabled: id !== null,
  });
}

export function useCreateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<MunkiPackage, ApiError, MunkiPackageCreateMutation>({
    mutationFn: (body) => unwrap(createMunkiPackage({ body })),
    onSuccess: (pkg) => {
      toast.success("Package created");
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll });
      void queryClient.invalidateQueries({
        queryKey: queryKeys.munkiSoftwareDetail(pkg.software_id),
      });
    },
  });
}

export function useUpdateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<MunkiPackage, ApiError, { id: number; body: MunkiPackageMutation }>({
    mutationFn: ({ id, body }) => unwrap(updateMunkiPackage({ path: { id }, body })),
    onSuccess: (pkg) => {
      toast.success("Package saved");
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackage(pkg.id) });
      void queryClient.invalidateQueries({
        queryKey: queryKeys.munkiSoftwareDetail(pkg.software_id),
      });
    },
  });
}

export function useBulkDeleteMunkiPackages() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteMunkiPackages({ body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
    },
  });
}
