import { queryOptions } from "@tanstack/react-query";

import type { ApiError, SessionBody } from "@/lib/api";
import { getSession, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type CurrentUser = NonNullable<SessionBody["user"]>;

export const sessionQueryOptions = queryOptions<SessionBody, ApiError>({
  queryKey: queryKeys.session,
  queryFn: async ({ signal }) => unwrap(getSession({ signal })),
});
