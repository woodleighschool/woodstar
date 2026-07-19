import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { ApiError, PageSoftwareTitle, SantaSoftwareReference } from "@/lib/api";
import { getSoftwareSantaReference, listSoftware, nullOn404, unwrap } from "@/lib/api";
import type { ListSoftwareData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { softwareTitleQueryOptions } from "@/lib/queries/software";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export type SoftwareListParams = NonNullable<ListSoftwareData["query"]>;

const SOFTWARE_REFRESH_MS = 30_000;
type RefetchOptions = { refetchInterval?: number | false };

export function useSoftware(params: SoftwareListParams = {}, options: RefetchOptions = {}) {
  const queryParams = {
    ...baseListParams(params),
    source: params.source && params.source.length > 0 ? params.source : undefined,
  };

  return useQuery<PageSoftwareTitle, ApiError>({
    queryKey: queryKeys.software(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listSoftware({
          query: queryParams,
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
    refetchInterval: options.refetchInterval,
  });
}

export function useSoftwareTitle(id: number | null, options: RefetchOptions = {}) {
  return useQuery(softwareTitleQueryOptions(id, options));
}

export function useSoftwareSantaReference(id: number | null) {
  return useQuery<SantaSoftwareReference | null, ApiError>({
    queryKey: queryKeys.softwareSantaReference(id),
    queryFn: ({ signal }) =>
      nullOn404(
        getSoftwareSantaReference({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
    refetchInterval: SOFTWARE_REFRESH_MS,
  });
}
