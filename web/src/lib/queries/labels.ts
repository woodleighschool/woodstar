import { queryOptions } from "@tanstack/react-query";

import type { ApiError, Label } from "@/lib/api";
import { getLabel, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export function labelQueryOptions(id: number | null) {
  return queryOptions<Label, ApiError>({
    queryKey: queryKeys.label(id),
    queryFn: ({ signal }) => unwrap(getLabel({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}
