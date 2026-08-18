import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { ApiError, PageActivityEvent } from "@lib/api";
import { listActivity, unwrap } from "@lib/api";
import type { ListActivityData } from "@lib/api-client/types.gen";

type ActivityParams = NonNullable<ListActivityData["query"]>;

const activityKeys = {
  all: ["activity"] as const,
  list: (params: ActivityParams) => ["activity", "list", params] as const,
};

export function useActivity(params: ActivityParams) {
  return useQuery<PageActivityEvent, ApiError>({
    queryKey: activityKeys.list(params),
    queryFn: ({ signal }) => unwrap(listActivity({ query: params, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: 30_000,
  });
}
