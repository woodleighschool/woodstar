import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";

import type { ApiError, LoginInputBody, SessionBody, SetupInputBody, User } from "@/lib/api";
import { completeSetup, createSession, deleteSession, getSession, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type { SessionBody };
export type CurrentUser = NonNullable<SessionBody["user"]>;

export function useSession(): { session: SessionBody | null; isLoading: boolean } {
  const { data, isLoading } = useQuery<SessionBody, ApiError>({
    queryKey: queryKeys.session,
    queryFn: async ({ signal }) => unwrap(getSession({ signal })),
  });

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
      await queryClient.invalidateQueries({ queryKey: queryKeys.session });
      await router.navigate({ to: "/login" });
    },
  });
}

export function useLogin() {
  const queryClient = useQueryClient();
  const router = useRouter();
  return useMutation<User, ApiError, LoginInputBody>({
    mutationFn: (body) => unwrap(createSession({ body })),
    meta: { inlineError: true },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.session });
      await router.navigate({ to: "/hosts" });
    },
  });
}

export function useSetup() {
  const queryClient = useQueryClient();
  const router = useRouter();
  return useMutation<User, ApiError, SetupInputBody>({
    mutationFn: (body) => unwrap(completeSetup({ body })),
    meta: { inlineError: true },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.session });
      await router.navigate({ to: "/hosts" });
    },
  });
}
