import { queryOptions } from "@tanstack/react-query";

import type { Account, ApiError } from "@lib/api";
import { getAccount, unwrap } from "@lib/api";

export const accountKey = ["account"] as const;

export const accountQueryOptions = queryOptions<Account, ApiError>({
  queryKey: accountKey,
  queryFn: async ({ signal }) => unwrap(getAccount({ signal })),
});
