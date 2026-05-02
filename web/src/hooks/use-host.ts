import { useQuery } from "@tanstack/react-query";

import { ApiError, fetchJson } from "@/lib/api";
import { endpoints } from "@/lib/endpoints";
import { queryKeys } from "@/lib/query-keys";
import type { HostDetail } from "@/lib/types";

export function useHost(id: string | undefined) {
  const endpoint = id ? endpoints.host(id) : null;
  const query = useQuery<HostDetail, ApiError>({
    queryKey: id ? queryKeys.host(id) : ["host", "_unset"],
    queryFn: () => fetchJson<HostDetail>(endpoint!.path),
    enabled: !!id && !!endpoint?.implemented,
    retry: 1,
  });

  return {
    data: query.data,
    query,
    isPending: !endpoint?.implemented,
  };
}
