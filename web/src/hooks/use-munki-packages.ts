import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

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
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

type MunkiPackageListParams = NonNullable<ListMunkiPackagesData["query"]>;

function packageQueryParams(params: MunkiPackageListParams) {
  return {
    ...baseListParams(params),
    type: params.type?.length ? params.type : undefined,
    software_id: params.software_id === 0 ? undefined : params.software_id,
  };
}

export function useMunkiPackages(params: MunkiPackageListParams = {}) {
  const query = packageQueryParams(params);
  return useQuery<PageMunkiPackage, ApiError>({
    queryKey: queryKeys.munkiPackages(query),
    queryFn: ({ signal }) => unwrap(listMunkiPackages({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiPackage(id: number | null) {
  return useQuery<MunkiPackage, ApiError>({
    queryKey: queryKeys.munkiPackage(id),
    queryFn: ({ signal }) => unwrap(getMunkiPackage({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function useCreateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<
    MunkiPackage,
    ApiError,
    { body: MunkiPackageCreateMutation; signal?: AbortSignal }
  >({
    mutationFn: ({ body, signal }) => unwrap(createMunkiPackage({ body, signal })),
    onSuccess: async (pkg) => {
      toast.success("Package created");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.munkiSoftwareDetail(pkg.software_id),
        }),
      ]);
    },
  });
}

export function useUpdateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<
    MunkiPackage,
    ApiError,
    { id: number; body: MunkiPackageMutation; signal?: AbortSignal }
  >({
    mutationFn: ({ id, body, signal }) =>
      unwrap(updateMunkiPackage({ path: { id }, body, signal })),
    onSuccess: async (pkg) => {
      toast.success("Package saved");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackage(pkg.id) }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.munkiSoftwareDetail(pkg.software_id),
        }),
      ]);
    },
  });
}

export function useBulkDeleteMunkiPackages() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteMunkiPackages({ body: { ids } })),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll }),
      ]);
    },
  });
}
