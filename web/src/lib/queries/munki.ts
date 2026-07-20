import { type QueryClient, queryOptions } from "@tanstack/react-query";

import type {
  ApiError,
  MunkiDistributionPointDetail,
  MunkiPackage,
  MunkiSoftwareDetail,
} from "@/lib/api";
import { getMunkiDistributionPoint, getMunkiPackage, getMunkiSoftware, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export async function invalidateMunkiSoftwareProjections(queryClient: QueryClient) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll }),
    queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll }),
    queryClient.invalidateQueries({ queryKey: queryKeys.munkiIconsAll }),
    queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll }),
  ]);
}

export function munkiSoftwareQueryOptions(id: number | null) {
  return queryOptions<MunkiSoftwareDetail, ApiError>({
    queryKey: queryKeys.munkiSoftwareDetail(id),
    queryFn: ({ signal }) => unwrap(getMunkiSoftware({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function munkiPackageQueryOptions(id: number | null) {
  return queryOptions<MunkiPackage, ApiError>({
    queryKey: queryKeys.munkiPackage(id),
    queryFn: ({ signal }) => unwrap(getMunkiPackage({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export interface MunkiDistributionPointRefreshOptions {
  staleTime?: number;
  refetchInterval?: number | false;
  refetchIntervalInBackground?: boolean;
}

export function munkiDistributionPointQueryOptions(
  id: number | null,
  refreshOptions: MunkiDistributionPointRefreshOptions = {},
) {
  return queryOptions<MunkiDistributionPointDetail, ApiError>({
    queryKey: queryKeys.munkiDistributionPoint(id),
    queryFn: ({ signal }) => unwrap(getMunkiDistributionPoint({ path: detailPath(id), signal })),
    enabled: id !== null,
    ...refreshOptions,
  });
}
