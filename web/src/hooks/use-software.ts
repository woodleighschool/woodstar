import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { ApiError, PageSoftwareTitle, SantaSoftwareReference, SoftwareTitle } from "@/lib/api";
import { getSoftware, getSoftwareSantaReference, listSoftware, nullOn404, unwrap } from "@/lib/api";
import type { ListSoftwareData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export type SoftwareListParams = NonNullable<ListSoftwareData["query"]>;

export function useSoftware(params: SoftwareListParams = {}) {
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
  });
}

export function useSoftwareTitle(id: number | null) {
  return useQuery<SoftwareTitle, ApiError>({
    queryKey: queryKeys.softwareTitle(id),
    queryFn: ({ signal }) =>
      unwrap(
        getSoftware({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
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
  });
}
