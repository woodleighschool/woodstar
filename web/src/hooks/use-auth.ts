import { useQuery } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type Session = Schemas["SessionBody"];
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
