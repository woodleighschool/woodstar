import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

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
  const queryParams = {
    q: nonEmpty(params.q),
    source: params.source && params.source.length > 0 ? params.source : undefined,
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: nonEmpty(params.order_key),
    order_direction: nonEmpty(params.order_direction),
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
