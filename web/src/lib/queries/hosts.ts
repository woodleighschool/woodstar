import { queryOptions } from "@tanstack/react-query";

import type { ApiError, HostDetail } from "@/lib/api";
import { getHost, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export function hostQueryOptions(
  id: number | null,
  options: { refetchInterval?: number | false } = {},
) {
  return queryOptions<HostDetail, ApiError>({
    queryKey: queryKeys.host(id),
    queryFn: ({ signal }) => unwrap(getHost({ path: detailPath(id), signal })),
    enabled: id !== null,
    refetchInterval: options.refetchInterval,
  });
}
