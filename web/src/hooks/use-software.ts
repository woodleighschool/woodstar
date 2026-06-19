import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type {
  ApiError,
  PageSoftwareTitle,
  SantaSoftwareReference,
  SoftwareTitle,
  SoftwareVersion,
} from "@/lib/api";
import { getSoftware, getSoftwareSantaReference, listSoftware, unwrap } from "@/lib/api";
import type { ListSoftwareData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";

export type { SoftwareTitle, SoftwareVersion };
export type SoftwareListResult = PageSoftwareTitle;
export type SoftwareSantaReference = SantaSoftwareReference;
export type SoftwareListParams = NonNullable<ListSoftwareData["query"]>;

export function useSoftware(params: SoftwareListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    source: params.source && params.source.length > 0 ? params.source : undefined,
  };

  return useQuery<SoftwareListResult, ApiError>({
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
          path: { id: id ?? 0 },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useSoftwareSantaReference(id: number | null) {
  return useQuery<SoftwareSantaReference, ApiError>({
    queryKey: queryKeys.softwareSantaReference(id),
    queryFn: ({ signal }) =>
      unwrap(
        getSoftwareSantaReference({
          path: { id: id ?? 0 },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}
