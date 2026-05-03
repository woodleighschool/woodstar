import { useQuery } from "@tanstack/react-query";

import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function useVersion() {
  return useQuery({
    queryKey: queryKeys.version,
    queryFn: ({ signal }) => unwrap(apiClient.GET("/version", { signal })),
    staleTime: 5 * 60_000,
    retry: 1,
  });
}
