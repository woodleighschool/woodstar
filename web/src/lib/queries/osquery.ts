import { queryOptions } from "@tanstack/react-query";

import type { ApiError, OsqueryCheck, OsqueryReport } from "@/lib/api";
import { getOsqueryCheck, getOsqueryReport, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export function checkQueryOptions(id: number | null) {
  return queryOptions<OsqueryCheck, ApiError>({
    queryKey: queryKeys.check(id),
    queryFn: ({ signal }) => unwrap(getOsqueryCheck({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function reportQueryOptions(id: number | null) {
  return queryOptions<OsqueryReport, ApiError>({
    queryKey: queryKeys.report(id),
    queryFn: ({ signal }) => unwrap(getOsqueryReport({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}
