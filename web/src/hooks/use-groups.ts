import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type { ApiError, Group, Page } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import type { ListGroupsData } from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { Group };
export type GroupListResult = Page<Group>;
export type GroupListParams = NonNullable<ListGroupsData["query"]>;

function groupQueryParams(params: GroupListParams = {}) {
  return {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
    values: params.values && params.values.length > 0 ? params.values : undefined,
  };
}

export function useGroups(params: GroupListParams = {}) {
  const queryParams = groupQueryParams(params);
  return useQuery<GroupListResult, ApiError>({
    queryKey: queryKeys.groups(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/groups", {
          params: { query: queryParams },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useGroup(id: number | null) {
  return useQuery<Group, ApiError>({
    queryKey: queryKeys.group(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/groups/{id}", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}
