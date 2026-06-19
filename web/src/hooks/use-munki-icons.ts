import { useQuery } from "@tanstack/react-query";

import type { ApiError, PageMunkiObjectView } from "@/lib/api";
import { listMunkiIcons, unwrap } from "@/lib/api";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";

export function useMunkiIcons(enabled = true) {
  const query = { page: 1, per_page: MAX_PAGE_SIZE };
  return useQuery<PageMunkiObjectView, ApiError>({
    queryKey: queryKeys.munkiIcons(query),
    queryFn: ({ signal }) => unwrap(listMunkiIcons({ query, signal })),
    enabled,
  });
}
