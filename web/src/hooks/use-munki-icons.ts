import { useQuery } from "@tanstack/react-query";

import type { ApiError, PageMunkiObjectView } from "@/lib/api";
import { listMunkiIcons, unwrap } from "@/lib/api";
import { baseListParams, MAX_PAGE_SIZE } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";

export function useMunkiIcons(enabled = true) {
  const query = baseListParams({}, { defaultPerPage: MAX_PAGE_SIZE });
  return useQuery<PageMunkiObjectView, ApiError>({
    queryKey: queryKeys.munkiIcons(query),
    queryFn: ({ signal }) => unwrap(listMunkiIcons({ query, signal })),
    enabled,
  });
}
