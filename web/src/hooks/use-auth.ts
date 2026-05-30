import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";

import type { ApiError, Session } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type { Session };
export type CurrentUser = NonNullable<Session["user"]>;

export function useSession(): { session: Session | null; isLoading: boolean } {
  const { data, isLoading } = useQuery<Session, ApiError>({
    queryKey: queryKeys.session,
    queryFn: async ({ signal }) => unwrap(apiClient.GET("/api/auth/session", { signal })),
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
    mutationFn: () => unwrap(apiClient.POST("/api/auth/logout")),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.session });
      await router.navigate({ to: "/login" });
    },
  });
}
