import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type SoftwareTitle = Schemas["SoftwareTitle"];
export type SoftwareListResult = Schemas["PaginatedBodySoftwareTitle"];
export type SoftwareVersion = Schemas["SoftwareVersion"];
export type SoftwareSantaReference = Schemas["SoftwareReference"];

export interface SoftwareListParams {
  q?: string;
  source?: string[];
  page_index?: number;
  page_size?: number;
  sort?: string;
}

export function useSoftware(params: SoftwareListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    source: params.source && params.source.length > 0 ? params.source : undefined,
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
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

export function useSoftwareTitle(id: number | null) {
  return useQuery<SoftwareTitle, ApiError>({
    queryKey: queryKeys.softwareTitle(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/software/{id}", {
          params: { path: { id: id ?? 0 } },
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
        apiClient.GET("/api/software/{id}/santa", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}
