import { keepPreviousData, queryOptions, useQuery } from "@tanstack/react-query";

import type { ApiError, PageSoftwareTitle, SoftwareTitle } from "@lib/api";
import { getSoftware, listSoftware, unwrap } from "@lib/api";
import type { ListSoftwareData } from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

type QueryParams = Record<string, unknown>;

export const softwareKeys = {
  all: ["software"] as const,
  list: (params?: QueryParams) => ["software", "list", params ?? {}] as const,
  detail: (id: number | null) => ["software", "detail", id] as const,
};

export type SoftwareListParams = NonNullable<ListSoftwareData["query"]>;

type RefetchOptions = { refetchInterval?: number | false };

export function softwareTitleQueryOptions(id: number | null, options: RefetchOptions = {}) {
  return queryOptions<SoftwareTitle, ApiError>({
    queryKey: softwareKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getSoftware({ path: detailPath(id), signal })),
    enabled: id !== null,
    refetchInterval: options.refetchInterval,
  });
}

export function useSoftware(params: SoftwareListParams = {}, options: RefetchOptions = {}) {
  const queryParams = {
    ...baseListParams(params),
    source: params.source && params.source.length > 0 ? params.source : undefined,
  };

  return useQuery<PageSoftwareTitle, ApiError>({
    queryKey: softwareKeys.list(queryParams),
    queryFn: ({ signal }) => unwrap(listSoftware({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: options.refetchInterval,
  });
}

export function useSoftwareTitle(id: number | null, options: RefetchOptions = {}) {
  return useQuery(softwareTitleQueryOptions(id, options));
}
