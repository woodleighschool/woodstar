import { queryOptions } from "@tanstack/react-query";

import type { ApiError, SoftwareTitle } from "@/lib/api";
import { getSoftware, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export function softwareTitleQueryOptions(
  id: number | null,
  options: { refetchInterval?: number | false } = {},
) {
  return queryOptions<SoftwareTitle, ApiError>({
    queryKey: queryKeys.softwareTitle(id),
    queryFn: ({ signal }) => unwrap(getSoftware({ path: detailPath(id), signal })),
    enabled: id !== null,
    refetchInterval: options.refetchInterval,
  });
}
