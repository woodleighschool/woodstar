import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";

import { firstAccessiblePath } from "@components/layout/nav-config";
import { accountQueryOptions } from "@features/account/query-options";
import type { ApiError, Principal, SessionBody, SessionCreateInputBody } from "@lib/api";
import { createSession, deleteSession, getSession, unwrap } from "@lib/api";

export type CurrentUser = NonNullable<SessionBody["user"]>;

export const sessionQueryOptions = queryOptions<SessionBody, ApiError>({
  queryKey: ["auth", "session"],
  queryFn: async ({ signal }) => unwrap(getSession({ signal })),
});

export function useSession(): { session: SessionBody | null; isLoading: boolean } {
  const { data, isLoading } = useQuery(sessionQueryOptions);

  return { session: data ?? null, isLoading };
}

export function useAuth(): { user: CurrentUser | null } {
  const { session } = useSession();
  return { user: session?.user ?? null };
}

export function useLogout() {
  const queryClient = useQueryClient();
  const router = useRouter();
  return useMutation({
    mutationFn: () => unwrap(deleteSession()),
    onSuccess: async () => {
      queryClient.clear();
      await router.navigate({ to: "/login" });
    },
  });
}

export function useLogin() {
  const queryClient = useQueryClient();
  const router = useRouter();
  return useMutation<Principal, ApiError, SessionCreateInputBody>({
    mutationFn: (body) => unwrap(createSession({ body })),
    meta: { inlineError: true },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: sessionQueryOptions.queryKey });
      const account = await queryClient.fetchQuery(accountQueryOptions);
      await router.navigate({ to: firstAccessiblePath(account) ?? "/account" });
    },
  });
}
