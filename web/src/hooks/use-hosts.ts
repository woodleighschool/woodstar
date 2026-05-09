import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type Host = Schemas["HostBody"];
export type HostSoftware = Schemas["HostSoftwareBody"];
export type HostListResult = Schemas["HostListBody"];
export type HostSoftwareListResult = Schemas["HostSoftwareListBody"];
export type HostQueriesResult = Schemas["HostReportsOutputBody"];
export type HostReport = Schemas["HostReportBody"];
export type HostQueryResultsResult = Schemas["HostQueryResultsOutputBody"];
export type HostChecksResult = Schemas["CheckHostsOutputBody"];

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
    q: nonEmpty(params.q),
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: nonEmpty(params.order_key),
    order_direction: nonEmpty(params.order_direction),
    status: nonEmpty(params.status),
    platform: nonEmpty(params.platform),
    label_id: nonEmpty(params.label_id),
    software_title_id: nonEmpty(params.software_title_id),
    software_id: nonEmpty(params.software_id),
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
    q: nonEmpty(params.q),
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 50,
    order_key: nonEmpty(params.order_key),
    order_direction: nonEmpty(params.order_direction),
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

export function useHostQueries(id: string) {
  return useQuery<HostQueriesResult, ApiError>({
    queryKey: queryKeys.hostQueries(id),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/hosts/{id}/queries", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useHostQueryResults(hostId: string, queryId: string) {
  return useQuery<HostQueryResultsResult, ApiError>({
    queryKey: queryKeys.hostQueryResults(hostId, queryId),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts/{id}/queries/{query_id}", {
          params: { path: { id: hostId, query_id: queryId } },
          signal,
        }),
      ),
    enabled: hostId !== "" && queryId !== "",
  });
}

export function useHostChecks(id: string) {
  return useQuery<HostChecksResult, ApiError>({
    queryKey: queryKeys.hostChecks(id),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/hosts/{id}/checks", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}
