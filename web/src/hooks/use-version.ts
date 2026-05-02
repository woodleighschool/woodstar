import { useQuery } from "@tanstack/react-query";

import { fetchJson } from "@/lib/api";
import { endpoints } from "@/lib/endpoints";
import { queryKeys } from "@/lib/query-keys";
import type { VersionInfo } from "@/lib/types";

export function useVersion() {
  return useQuery({
    queryKey: queryKeys.version,
    queryFn: () => fetchJson<VersionInfo>(endpoints.version.path),
    staleTime: 5 * 60_000,
    retry: 1,
  });
}
