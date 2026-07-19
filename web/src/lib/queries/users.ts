import { queryOptions } from "@tanstack/react-query";

import type { ApiError, User } from "@/lib/api";
import { getUser, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export function userQueryOptions(id: number | null) {
  return queryOptions<User, ApiError>({
    queryKey: queryKeys.user(id),
    queryFn: ({ signal }) => unwrap(getUser({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}
