import { useQuery } from "@tanstack/react-query";

import { MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type { ApiError, MunkiObjectView } from "@/lib/api";
import { listMunkiIcons, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

interface MunkiIconsPage {
  items: MunkiObjectView[];
  count: number;
}

export function useMunkiIcons(enabled = true) {
  const query = { page: 1, per_page: MAX_PAGE_SIZE };
  return useQuery<MunkiIconsPage, ApiError>({
    queryKey: queryKeys.munkiIcons(query),
    queryFn: ({ signal }) => unwrap(listMunkiIcons({ query, signal })),
    enabled,
  });
}
