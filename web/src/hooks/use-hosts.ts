import { useQuery } from "@tanstack/react-query";

import { ApiError, apiClient, type Schemas, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type Host = Schemas["HostBody"];
export type HostSoftware = Schemas["HostSoftwareBody"];

export function useHosts() {
  return useQuery<Host[], ApiError>({
    queryKey: queryKeys.hosts,
    queryFn: async ({ signal }) =>
      (await unwrap(apiClient.GET("/api/hosts", { signal }))) ?? [],
  });
}

export function useHost(id: string) {
  return useQuery<Host, ApiError>({
    queryKey: queryKeys.host(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts/{id}", {
          params: { path: { id } },
          signal,
        }),
      ),
    enabled: id !== "",
  });
}

export function useHostSoftware(id: string) {
  return useQuery<HostSoftware[], ApiError>({
    queryKey: queryKeys.hostSoftware(id),
    queryFn: async ({ signal }) =>
      (await unwrap(
        apiClient.GET("/api/hosts/{id}/software", {
          params: { path: { id } },
          signal,
        }),
      )) ?? [],
    enabled: id !== "",
  });
}
