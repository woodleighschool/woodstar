import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { ApiError, Group, PageGroup } from "@/lib/api";
import { getGroup, listGroups, unwrap } from "@/lib/api";
import type { ListGroupsData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export type GroupListParams = NonNullable<ListGroupsData["query"]>;

function groupQueryParams(params: GroupListParams = {}) {
  return {
    ...baseListParams(params),
    values: params.values && params.values.length > 0 ? params.values : undefined,
  };
}

export function useGroups(params: GroupListParams = {}) {
  const queryParams = groupQueryParams(params);
  return useQuery<PageGroup, ApiError>({
    queryKey: queryKeys.groups(queryParams),
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
    queryKey: queryKeys.group(id),
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
