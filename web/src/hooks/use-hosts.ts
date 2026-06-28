import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type {
  ApiError,
  HostDetail,
  MunkiHostState,
  OsqueryCheckHostStatus,
  OsqueryHostReport,
  PageHost,
  PageHostSoftware,
  PageRuleStatus,
  SantaHostState,
} from "@/lib/api";
import {
  bulkDeleteHosts,
  clearHostPrimaryUser,
  deleteHost,
  getHost,
  getHostMunkiState,
  getHostSantaState,
  listHostOsqueryChecks,
  listHostOsqueryReports,
  listHosts,
  listHostSantaRules,
  listHostSoftware,
  nullOn404,
  setHostPrimaryUser,
  unwrap,
} from "@/lib/api";
import type {
  ListHostSantaRulesData,
  ListHostsData,
  ListHostSoftwareData,
} from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";
import { nonEmpty } from "@/lib/utils";

const HOST_SANTA_RULES_PAGE_SIZE = 100;
type HostSantaRulesParams = NonNullable<ListHostSantaRulesData["query"]>;

interface HostPrimaryUserMutation {
  email: string;
}

type HostListParams = NonNullable<ListHostsData["query"]>;

export function useHosts(params: HostListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    status: nonEmpty(params.status),
    label_id: params.label_id,
    software_title_id: params.software_title_id,
    software_id: params.software_id,
    ids: params.ids && params.ids.length > 0 ? params.ids : undefined,
  };

  return useQuery<PageHost, ApiError>({
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
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useHostMunkiState(id: number | null) {
  return useQuery<MunkiHostState | null, ApiError>({
    queryKey: queryKeys.hostMunkiState(id),
    queryFn: ({ signal }) => nullOn404(getHostMunkiState({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function useHostSantaState(id: number | null) {
  return useQuery<SantaHostState | null, ApiError>({
    queryKey: queryKeys.hostSantaState(id),
    queryFn: ({ signal }) => nullOn404(getHostSantaState({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function useDeleteHost() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteHost({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

export function useBulkDeleteHosts() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteHosts({ body: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

export function useSetHostPrimaryUser() {
  const queryClient = useQueryClient();
  return useMutation<HostDetail, ApiError, { id: number; body: HostPrimaryUserMutation }>({
    mutationFn: ({ id, body }) => unwrap(setHostPrimaryUser({ path: { id }, body })),
    onSuccess: async (host) => {
      toast.success("Primary user set");
      queryClient.setQueryData(queryKeys.host(host.id), host);
      await queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

export function useClearHostPrimaryUser() {
  const queryClient = useQueryClient();
  return useMutation<HostDetail, ApiError, number>({
    mutationFn: (id) => unwrap(clearHostPrimaryUser({ path: { id } })),
    onSuccess: async (host) => {
      toast.success("Primary user cleared");
      queryClient.setQueryData(queryKeys.host(host.id), host);
      await queryClient.invalidateQueries({ queryKey: queryKeys.hostsAll });
    },
  });
}

type HostSoftwareListParams = NonNullable<ListHostSoftwareData["query"]>;

export function useHostSoftware(id: number | null, params: HostSoftwareListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    source: params.source && params.source.length > 0 ? params.source : undefined,
  };

  return useQuery<PageHostSoftware, ApiError>({
    queryKey: queryKeys.hostSoftware(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listHostSoftware({
          path: detailPath(id),
          query: queryParams,
          signal,
        }),
      ),
    enabled: id !== null,
    placeholderData: keepPreviousData,
  });
}

export function useHostOsqueryReports(id: number | null) {
  return useQuery<OsqueryHostReport[], ApiError>({
    queryKey: queryKeys.hostOsqueryReports(id),
    queryFn: ({ signal }) =>
      unwrap(
        listHostOsqueryReports({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useHostOsqueryChecks(id: number | null) {
  return useQuery<OsqueryCheckHostStatus[], ApiError>({
    queryKey: queryKeys.hostOsqueryChecks(id),
    queryFn: ({ signal }) =>
      unwrap(
        listHostOsqueryChecks({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useHostSantaRules(id: number | null, params: HostSantaRulesParams = {}) {
  const queryParams = {
    ...baseListParams(params, { defaultPerPage: HOST_SANTA_RULES_PAGE_SIZE }),
  };

  return useQuery<PageRuleStatus, ApiError>({
    queryKey: queryKeys.hostSantaRules(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listHostSantaRules({
          path: detailPath(id),
          query: queryParams,
          signal,
        }),
      ),
    enabled: id !== null,
    placeholderData: keepPreviousData,
  });
}
