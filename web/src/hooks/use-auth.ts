import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";

import type { ApiError, Principal, SessionBody, SessionCreateInputBody } from "@/lib/api";
import { createSession, deleteSession, unwrap } from "@/lib/api";
import { type CurrentUser, sessionQueryOptions } from "@/lib/session";

export type { CurrentUser };

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
      await router.navigate({ to: "/hosts" });
    },
  });
}
