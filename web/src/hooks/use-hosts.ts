import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import type {
  ApiError,
  Host,
  HostDetail,
  OsqueryCheckHostStatus,
  OsqueryHostReport,
  PageHost,
  PageHostSoftwareRow,
  PageRuleStatus,
} from "@/lib/api";
import {
  bulkDeleteHosts,
  deleteHost,
  deleteHostUserAffinity,
  getHost,
  listHostOsqueryChecks,
  listHostOsqueryReports,
  listHosts,
  listHostSantaRules,
  listHostSoftware,
  putHostUserAffinity,
  unwrap,
} from "@/lib/api";
import type {
  ListHostSantaRulesData,
  ListHostsData,
  ListHostSoftwareData,
} from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { Host, HostDetail, OsqueryHostReport };

const HOST_SANTA_RULES_PAGE_SIZE = 100;

type HostListResult = PageHost;
type HostSoftwareListResult = PageHostSoftwareRow;
type HostReportsResult = OsqueryHostReport[];
type HostChecksResult = OsqueryCheckHostStatus[];
type HostSantaRulesResult = PageRuleStatus;
type HostSantaRulesParams = NonNullable<ListHostSantaRulesData["query"]>;

interface HostUserAffinityMutation {
  email: string;
}

type HostListParams = NonNullable<ListHostsData["query"]>;

export function useHosts(params: HostListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
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
        listHosts({
          query: queryParams,
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
        getHost({
          path: { id: id ?? 0 },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useDeleteHost() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteHost({ path: { id } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

export function useBulkDeleteHosts() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteHosts({ body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

export function useSetHostUserAffinity() {
  const queryClient = useQueryClient();
  return useMutation<HostDetail, ApiError, { id: number; body: HostUserAffinityMutation }>({
    mutationFn: ({ id, body }) => unwrap(putHostUserAffinity({ path: { id }, body })),
    onSuccess: async (host) => {
      toast.success("User affinity set");
      queryClient.setQueryData(queryKeys.host(host.id), host);
      await queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

export function useClearHostUserAffinity() {
  const queryClient = useQueryClient();
  return useMutation<HostDetail, ApiError, number>({
    mutationFn: (id) => unwrap(deleteHostUserAffinity({ path: { id } })),
    onSuccess: async (host) => {
      toast.success("User affinity cleared");
      queryClient.setQueryData(queryKeys.host(host.id), host);
      await queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

type HostSoftwareListParams = NonNullable<ListHostSoftwareData["query"]>;

export function useHostSoftware(id: number | null, params: HostSoftwareListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
    source: params.source && params.source.length > 0 ? params.source : undefined,
  };

  return useQuery<HostSoftwareListResult, ApiError>({
    queryKey: queryKeys.hostSoftware(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listHostSoftware({
          path: { id: id ?? 0 },
          query: queryParams,
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
        listHostOsqueryReports({
          path: { id: id ?? 0 },
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
        listHostOsqueryChecks({
          path: { id: id ?? 0 },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useHostSantaRules(id: number | null, params: HostSantaRulesParams = {}) {
  const queryParams = {
    page: params.page ?? 1,
    per_page: params.per_page ?? HOST_SANTA_RULES_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };

  return useQuery<HostSantaRulesResult, ApiError>({
    queryKey: queryKeys.hostSantaRules(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listHostSantaRules({
          path: { id: id ?? 0 },
          query: queryParams,
          signal,
        }),
      ),
    enabled: id !== null,
    placeholderData: keepPreviousData,
  });
}
