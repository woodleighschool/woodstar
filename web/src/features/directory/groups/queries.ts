import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { ApiError, Group, PageGroup } from "@lib/api";
import { getGroup, listGroups, unwrap } from "@lib/api";
import type { ListGroupsData } from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

export type GroupListParams = NonNullable<ListGroupsData["query"]>;

type QueryParams = Record<string, unknown>;

export const groupKeys = {
  all: ["groups"] as const,
  list: (params?: QueryParams) => ["groups", "list", params ?? {}] as const,
  detail: (id: number | null) => ["groups", "detail", id] as const,
};

function groupQueryParams(params: GroupListParams = {}) {
  return {
    ...baseListParams(params),
    values: params.values && params.values.length > 0 ? params.values : undefined,
  };
}

export function useGroups(params: GroupListParams = {}) {
  const queryParams = groupQueryParams(params);
  return useQuery<PageGroup, ApiError>({
    queryKey: groupKeys.list(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listGroups({
          query: queryParams,
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useGroup(id: number | null) {
  return useQuery<Group, ApiError>({
    queryKey: groupKeys.detail(id),
    queryFn: ({ signal }) =>
      unwrap(
        getGroup({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
}
