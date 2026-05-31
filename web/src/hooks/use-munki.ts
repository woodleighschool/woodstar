import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type {
  ApiError,
  MunkiAssignment,
  MunkiAssignmentPage,
  MunkiRelease,
  MunkiReleasePage,
  MunkiSoftwareTitle,
  MunkiSoftwareTitlePage,
} from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { MunkiAssignment, MunkiRelease, MunkiSoftwareTitle };
export type MunkiListResult<T> = {
  items: T[] | null;
  count: number;
};

export interface MunkiListParams {
  q?: string;
  page_index?: number;
  page_size?: number;
  sort?: string;
}

function queryParams(params: MunkiListParams) {
  return {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
  };
}

export function useMunkiSoftwareTitles(params: MunkiListParams = {}) {
  const query = queryParams(params);
  return useQuery<MunkiSoftwareTitlePage, ApiError>({
    queryKey: queryKeys.munkiSoftwareTitles(query),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/software-titles", {
          params: { query },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiReleases(params: MunkiListParams = {}) {
  const query = queryParams(params);
  return useQuery<MunkiReleasePage, ApiError>({
    queryKey: queryKeys.munkiReleases(query),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/releases", {
          params: { query },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiAssignments(params: MunkiListParams = {}) {
  const query = queryParams(params);
  return useQuery<MunkiAssignmentPage, ApiError>({
    queryKey: queryKeys.munkiAssignments(query),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/munki/assignments", {
          params: { query },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}
