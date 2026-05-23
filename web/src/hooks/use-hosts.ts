import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type Host = Schemas["Host"];
export type HostDetail = Schemas["HostDetailBody"];
export type HostSoftware = Schemas["HostSoftwareRow"];
export type HostListResult = Schemas["PaginatedBodyHost"];
export type HostSoftwareListResult = Schemas["PaginatedBodyHostSoftwareRow"];
export type HostReportsResult = Schemas["HostReportsBody"];
export type HostReport = Schemas["HostReport"];
export type HostReportResultsResult = Schemas["HostReportResultsBody"];
export type HostChecksResult = Schemas["CheckHostsOutputBody"];
export type HostSantaEffectiveRulesResult = Schemas["PaginatedBodyEffectiveRuleStatus"];
export type HostSantaEffectiveRule = Schemas["EffectiveRuleStatus"];

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
  return useQuery<HostDetail, ApiError>({
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

export function useDeleteHost() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/hosts/{id}", { params: { path: { id: String(id) } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["hosts"] });
    },
  });
}

export function useBulkDeleteHosts() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(apiClient.POST("/api/hosts/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["hosts"] });
    },
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

export function useHostReports(id: string) {
  return useQuery<HostReportsResult, ApiError>({
    queryKey: queryKeys.hostReports(id),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/hosts/{id}/osquery/reports", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useHostReportResults(hostId: string, reportId: string) {
  return useQuery<HostReportResultsResult, ApiError>({
    queryKey: queryKeys.hostReportResults(hostId, reportId),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts/{id}/osquery/reports/{report_id}", {
          params: { path: { id: hostId, report_id: reportId } },
          signal,
        }),
      ),
    enabled: hostId !== "" && reportId !== "",
  });
}

export function useHostChecks(id: string) {
  return useQuery<HostChecksResult, ApiError>({
    queryKey: queryKeys.hostChecks(id),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/hosts/{id}/osquery/checks", { params: { path: { id } }, signal })),
    enabled: id !== "",
  });
}

export function useHostSantaEffectiveRules(id: string, params: ListParams = {}) {
  const queryParams = {
    page: Math.max(1, params.page ?? 1),
    per_page: params.per_page ?? 100,
    order_key: nonEmpty(params.order_key),
    order_direction: nonEmpty(params.order_direction),
  };

  return useQuery<HostSantaEffectiveRulesResult, ApiError>({
    queryKey: queryKeys.hostSantaEffectiveRules(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts/{id}/santa/effective-rules", {
          params: { path: { id }, query: queryParams },
          signal,
        }),
      ),
    enabled: id !== "",
    placeholderData: keepPreviousData,
  });
}
