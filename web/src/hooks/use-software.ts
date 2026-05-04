import { useQuery } from "@tanstack/react-query";

import { ApiError, apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type SoftwareTitle = Schemas["SoftwareTitleBody"];

export function useSoftware() {
  return useQuery<SoftwareTitle[], ApiError>({
    queryKey: queryKeys.software,
    queryFn: async ({ signal }) => (await unwrap(apiClient.GET("/api/software", { signal }))) ?? [],
  });
}
