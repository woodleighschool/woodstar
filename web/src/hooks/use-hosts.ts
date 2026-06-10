import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  ApiError,
  CheckHostStatus,
  Host,
  HostDetail,
  HostReport,
  HostSoftwareRow,
  Page,
  RuleStatus,
} from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import type { ListHostSantaRulesData } from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { Host, HostDetail, HostReport };
export type HostSoftware = HostSoftwareRow;
export type HostSantaRule = RuleStatus;

type HostListResult = Page<Host>;
type HostSoftwareListResult = Page<HostSoftwareRow>;
type HostReportsResult = HostReport[];
type HostChecksResult = CheckHostStatus[];
type HostSantaRulesResult = Page<RuleStatus>;
type HostSantaRulesParams = NonNullable<ListHostSantaRulesData["query"]>;

interface HostUserAffinityMutation {
  email: string;
}

interface ListParams {
  q?: string;
  page_index?: number;
  page_size?: number;
  sort?: string;
}

interface HostListParams extends ListParams {
  status?: string;
  label_id?: number;
  software_title_id?: number;
  software_id?: number;
  ids?: number[];
  check_id?: number;
  check_response?: "pass" | "fail";
}

export function useHosts(params: HostListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
    status: nonEmpty(params.status),
    label_id: params.label_id,
    software_title_id: params.software_title_id,
    software_id: params.software_id,
    ids: params.ids && params.ids.length > 0 ? params.ids : undefined,
    check_id: params.check_id,
    check_response: params.check_response,
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

export function useHost(id: number | null) {
  return useQuery<HostDetail, ApiError>({
    queryKey: queryKeys.host(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts/{id}", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useDeleteHost() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/hosts/{id}", { params: { path: { id } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

export function useBulkDeleteHosts() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(apiClient.POST("/api/hosts/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

export function useSetHostUserAffinity() {
  const queryClient = useQueryClient();
  return useMutation<HostDetail, ApiError, { id: number; body: HostUserAffinityMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(apiClient.PUT("/api/hosts/{id}/user-affinity", { params: { path: { id } }, body })),
    onSuccess: async (host) => {
      queryClient.setQueryData(queryKeys.host(host.id), host);
      await queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
    meta: { inlineError: true },
  });
}

export function useClearHostUserAffinity() {
  const queryClient = useQueryClient();
  return useMutation<HostDetail, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/hosts/{id}/user-affinity", { params: { path: { id } } })),
    onSuccess: async (host) => {
      queryClient.setQueryData(queryKeys.host(host.id), host);
      await queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
    meta: { inlineError: true },
  });
}

interface HostSoftwareListParams extends ListParams {
  source?: string[];
}

export function useHostSoftware(id: number | null, params: HostSoftwareListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
    source: params.source && params.source.length > 0 ? params.source : undefined,
  };

  return useQuery<HostSoftwareListResult, ApiError>({
    queryKey: queryKeys.hostSoftware(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts/{id}/software", {
          params: { path: { id: id ?? 0 }, query: queryParams },
          signal,
        }),
      ),
    enabled: id !== null,
    placeholderData: keepPreviousData,
  });
}

export function useHostReports(id: number | null) {
  return useQuery<HostReportsResult, ApiError>({
    queryKey: queryKeys.hostReports(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts/{id}/osquery/reports", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useHostChecks(id: number | null) {
  return useQuery<HostChecksResult, ApiError>({
    queryKey: queryKeys.hostChecks(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts/{id}/osquery/checks", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useHostSantaRules(id: number | null, params: HostSantaRulesParams = {}) {
  const queryParams = {
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 100,
    sort: nonEmpty(params.sort),
  };

  return useQuery<HostSantaRulesResult, ApiError>({
    queryKey: queryKeys.hostSantaRules(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/hosts/{id}/santa/rules", {
          params: { path: { id: id ?? 0 }, query: queryParams },
          signal,
        }),
      ),
    enabled: id !== null,
    placeholderData: keepPreviousData,
  });
}
