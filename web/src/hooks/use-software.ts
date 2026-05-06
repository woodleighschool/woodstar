import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { ApiError, apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type SoftwareTitle = Schemas["SoftwareTitleBody"];
export type SoftwareListResult = Schemas["SoftwareListBody"];
export type SoftwareVersion = Schemas["SoftwareVersionBody"];
export type SoftwareTitleResult = Schemas["SoftwareGetBody"];

export interface SoftwareListParams {
  q?: string;
  source?: string[];
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: string;
}

export function useSoftware(params: SoftwareListParams = {}) {
  const sourceParam = params.source && params.source.length > 0 ? params.source : undefined;
  const qParam = params.q?.trim() || undefined;
  const pageParam = Math.max(0, params.page ?? 0);
  const perPageParam = params.per_page ?? 50;
  const queryParams = {
    q: qParam,
    source: sourceParam,
    page: pageParam,
    per_page: perPageParam,
    order_key: params.order_key || undefined,
    order_direction: params.order_direction || undefined,
  };

  return useQuery<SoftwareListResult, ApiError>({
    queryKey: queryKeys.software(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/software", {
          params: { query: queryParams },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useSoftwareTitle(id: string) {
  return useQuery<SoftwareTitleResult, ApiError>({
    queryKey: queryKeys.softwareTitle(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/software/{id}", {
          params: { path: { id } },
          signal,
        }),
      ),
    enabled: id !== "",
  });
}
