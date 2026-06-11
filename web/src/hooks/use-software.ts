import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type { ApiError, Page, SoftwareReference, SoftwareTitle, SoftwareVersion } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { SoftwareTitle, SoftwareVersion };
export type SoftwareListResult = Page<SoftwareTitle>;
export type SoftwareSantaReference = SoftwareReference;

export interface SoftwareListParams {
  q?: string;
  source?: string[];
  page?: number;
  per_page?: number;
  sort?: string;
}

export function useSoftware(params: SoftwareListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    source: params.source && params.source.length > 0 ? params.source : undefined,
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
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
