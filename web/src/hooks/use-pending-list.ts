import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import { ApiError, fetchJson } from "@/lib/api";
import type { EndpointSpec } from "@/lib/endpoints";

export interface PendingListResult<T> {
  data: T[];
  query: UseQueryResult<T[], ApiError>;
  isPending: boolean;
}

/**
 * useResourceList wraps useQuery with the implemented flag so unimplemented
 * endpoints surface a stable "pending" state instead of triggering 404s.
 */
export function useResourceList<T>(
  endpoint: EndpointSpec,
  queryKey: readonly unknown[],
): PendingListResult<T> {
  const query = useQuery<T[], ApiError>({
    queryKey: [...queryKey],
    queryFn: () => fetchJson<T[]>(endpoint.path),
    enabled: endpoint.implemented,
    retry: 1,
  });

  return {
    data: query.data ?? [],
    query,
    isPending: !endpoint.implemented,
  };
}
