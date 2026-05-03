import { useQuery } from "@tanstack/react-query";

import { ApiError, apiClient, type Schemas, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type CurrentUser = Schemas["UserBody"];

export function useAuth(): { user: CurrentUser | null } {
  const { data } = useQuery<CurrentUser | null, ApiError>({
    queryKey: queryKeys.authMe,
    queryFn: async ({ signal }) => {
      const result = await apiClient.GET("/api/auth/me", { signal });
      if (result.response.status === 401) {
        return null;
      }
      return unwrap(Promise.resolve(result));
    },
    retry: false,
    staleTime: 30_000,
  });

  return { user: data ?? null };
}
