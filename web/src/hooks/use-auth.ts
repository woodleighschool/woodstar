import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";

import type { ApiError, LoginInput, Session, SetupInput, User } from "@/lib/api";
import { completeSetup, createSession, deleteSession, getSession, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type { Session };
export type CurrentUser = NonNullable<Session["user"]>;

export function useSession(): { session: Session | null; isLoading: boolean } {
  const { data, isLoading } = useQuery<Session, ApiError>({
    queryKey: queryKeys.session,
    queryFn: async ({ signal }) => unwrap(getSession({ signal })),
    retry: false,
    staleTime: 30_000,
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
  return useMutation<User, ApiError, LoginInput>({
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
  return useMutation<User, ApiError, SetupInput>({
    mutationFn: (body) => unwrap(completeSetup({ body })),
    meta: { inlineError: true },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.session });
      await router.navigate({ to: "/hosts" });
    },
  });
}
