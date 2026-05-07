import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { ApiError, apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type Host = Schemas["HostBody"];
export type HostSoftware = Schemas["HostSoftwareBody"];
export type HostListResult = Schemas["HostListBody"];
export type HostSoftwareListResult = Schemas["HostSoftwareListBody"];

export interface ListParams {
  q?: string;
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: string;
}

export interface HostListParams extends ListParams {
  status?: string;
  platform?: string;
  label_id?: string;
  software_title_id?: string;
  software_id?: string;
}

export function useHosts(params: HostListParams = {}) {
  const queryParams = {
    q: params.q?.trim() || undefined,
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: params.order_key || undefined,
    order_direction: params.order_direction || undefined,
    status: params.status || undefined,
    platform: params.platform || undefined,
    label_id: params.label_id || undefined,
    software_title_id: params.software_title_id || undefined,
    software_id: params.software_id || undefined,
  };

  return useQuery<HostListResult, ApiError>({
    queryKey: queryKeys.hosts(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts", {
          params: { query: queryParams },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
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

export interface HostSoftwareListParams extends ListParams {
  source?: string[];
}

export function useHostSoftware(id: string, params: HostSoftwareListParams = {}) {
  const queryParams = {
    q: params.q?.trim() || undefined,
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: params.order_key || undefined,
    order_direction: params.order_direction || undefined,
    source: params.source && params.source.length > 0 ? params.source : undefined,
  };

  return useQuery<HostSoftwareListResult, ApiError>({
    queryKey: queryKeys.hostSoftware(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts/{id}/software", {
          params: { path: { id }, query: queryParams },
          signal,
        }),
      ),
    enabled: id !== "",
    placeholderData: keepPreviousData,
  });
}
