import { useQuery } from "@tanstack/react-query";

import { ApiError, fetchJson } from "@/lib/api";
import { endpoints } from "@/lib/endpoints";
import { queryKeys } from "@/lib/query-keys";
import type { CurrentUser } from "@/lib/types";

export interface AuthState {
  user: CurrentUser | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  isPending: boolean;
}

export function useAuth(): AuthState {
  const { data, error, isLoading, fetchStatus } = useQuery<
    CurrentUser | null,
    ApiError
  >({
    queryKey: queryKeys.authMe,
    queryFn: async () => {
      try {
        return await fetchJson<CurrentUser>(endpoints.authMe.path);
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          return null;
        }
        throw err;
      }
    },
    enabled: endpoints.authMe.implemented,
    retry: false,
    staleTime: 30_000,
  });

  if (!endpoints.authMe.implemented) {
    return { user: null, isLoading: false, isAuthenticated: false, isPending: true };
  }

  const user = data ?? null;
  const isAuthenticated = !!user && !error;
  const loading = isLoading && fetchStatus !== "idle";

  return { user, isLoading: loading, isAuthenticated, isPending: false };
}
