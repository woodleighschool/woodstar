import {
  keepPreviousData,
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import type {
  ApiError,
  Host,
  HostDetail,
  MunkiHostState,
  OsqueryReportSnapshot,
  PageCheckHostStatus,
  PageHost,
  PageHostManifestSoftware,
  PageHostSoftware,
  PageReportSnapshot,
  PageRuleStatus,
  SantaHostState,
} from "@lib/api";
import {
  bulkDeleteHosts,
  clearHostPrimaryUser,
  getHost,
  getHostMunkiState,
  getHostSantaState,
  listHostMunkiSoftware,
  listHostOsqueryChecks,
  listHostOsqueryReports,
  listHosts,
  listHostSantaRules,
  listHostSoftware,
  nullOn404,
  setHostPrimaryUser,
  unwrap,
} from "@lib/api";
import type {
  ListHostMunkiSoftwareData,
  ListHostOsqueryChecksData,
  ListHostOsqueryReportsData,
  ListHostSantaRulesData,
  ListHostSoftwareData,
  ListHostsData,
} from "@lib/api-client/types.gen";
import { baseListParams, collectAllPages } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

type QueryParams = Record<string, unknown>;

export const hostKeys = {
  all: ["hosts"] as const,
  list: (params?: QueryParams) => ["hosts", "list", params ?? {}] as const,
  detail: (id: number | null) => ["hosts", "detail", id] as const,
  software: (id: number | null, params?: QueryParams) =>
    ["hosts", "detail", id, "software", "list", params ?? {}] as const,
  munkiState: (id: number | null) => ["hosts", "detail", id, "munki"] as const,
  munkiSoftware: (id: number | null, params?: QueryParams) =>
    ["hosts", "detail", id, "munki", "software", "list", params ?? {}] as const,
  osqueryReports: (id: number | null, params?: QueryParams) =>
    ["hosts", "detail", id, "osquery", "reports", params ?? {}] as const,
  osqueryChecks: (id: number | null, params?: QueryParams) =>
    ["hosts", "detail", id, "osquery", "checks", params ?? {}] as const,
  santaState: (id: number | null) => ["hosts", "detail", id, "santa"] as const,
  santaRules: (id: number | null, params?: QueryParams) =>
    ["hosts", "detail", id, "santa", "rules", "list", params ?? {}] as const,
};

const HOST_SANTA_RULES_PAGE_SIZE = 100;
const HOST_REFRESH_MS = 30_000;

type HostListParams = NonNullable<ListHostsData["query"]>;
type HostSoftwareListParams = NonNullable<ListHostSoftwareData["query"]>;
type HostMunkiSoftwareParams = NonNullable<ListHostMunkiSoftwareData["query"]>;
type HostOsqueryChecksParams = NonNullable<ListHostOsqueryChecksData["query"]>;
type HostOsqueryReportsParams = NonNullable<ListHostOsqueryReportsData["query"]>;
type HostSantaRulesParams = NonNullable<ListHostSantaRulesData["query"]>;
type RefetchOptions = { refetchInterval?: number | false };

interface HostPrimaryUserMutation {
  email: string;
}

function hostListQueryParams(params: HostListParams) {
  return {
    ...baseListParams(params),
    status: params.status,
    label_id: params.label_id,
    software_title_id: params.software_title_id,
    software_id: params.software_id,
    ids: params.ids && params.ids.length > 0 ? params.ids : undefined,
  };
}

export function hostQueryOptions(id: number | null, options: RefetchOptions = {}) {
  return queryOptions<HostDetail, ApiError>({
    queryKey: hostKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getHost({ path: detailPath(id), signal })),
    enabled: id !== null,
    refetchInterval: options.refetchInterval,
  });
}

export function useHosts(params: HostListParams = {}, options: RefetchOptions = {}) {
  const queryParams = hostListQueryParams(params);

  return useQuery<PageHost, ApiError>({
    queryKey: hostKeys.list(queryParams),
    queryFn: ({ signal }) => unwrap(listHosts({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: options.refetchInterval,
  });
}

export async function listAllHosts(params: HostListParams = {}): Promise<Host[]> {
  return collectAllPages((page, perPage) =>
    unwrap(
      listHosts({
        query: hostListQueryParams({ ...params, page, per_page: perPage }),
      }),
    ),
  );
}

export function useHost(id: number | null, options: RefetchOptions = {}) {
  return useQuery(hostQueryOptions(id, options));
}

export function useHostMunkiState(id: number | null) {
  return useQuery<MunkiHostState | null, ApiError>({
    queryKey: hostKeys.munkiState(id),
    queryFn: ({ signal }) => nullOn404(getHostMunkiState({ path: detailPath(id), signal })),
    enabled: id !== null,
    refetchInterval: HOST_REFRESH_MS,
  });
}

export function useHostMunkiSoftware(id: number | null, params: HostMunkiSoftwareParams = {}) {
  const queryParams = baseListParams(params);
  return useQuery<PageHostManifestSoftware, ApiError>({
    queryKey: hostKeys.munkiSoftware(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listHostMunkiSoftware({
          path: detailPath(id),
          query: queryParams,
          signal,
        }),
      ),
    enabled: id !== null,
    placeholderData: keepPreviousData,
    refetchInterval: HOST_REFRESH_MS,
  });
}

export function useHostSantaState(id: number | null) {
  return useQuery<SantaHostState | null, ApiError>({
    queryKey: hostKeys.santaState(id),
    queryFn: ({ signal }) => nullOn404(getHostSantaState({ path: detailPath(id), signal })),
    enabled: id !== null,
    refetchInterval: HOST_REFRESH_MS,
  });
}

export function useBulkDeleteHosts() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteHosts({ query: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: hostKeys.all });
    },
  });
}

export function useSetHostPrimaryUser() {
  const queryClient = useQueryClient();
  return useMutation<HostDetail, ApiError, { id: number; body: HostPrimaryUserMutation }>({
    mutationFn: ({ id, body }) => unwrap(setHostPrimaryUser({ path: { id }, body })),
    onSuccess: async (host) => {
      toast.add({ title: "Primary user set", type: "success" });
      queryClient.setQueryData(hostKeys.detail(host.id), host);
      await queryClient.invalidateQueries({ queryKey: hostKeys.all });
    },
  });
}

export function useClearHostPrimaryUser() {
  const queryClient = useQueryClient();
  return useMutation<HostDetail, ApiError, number>({
    mutationFn: (id) => unwrap(clearHostPrimaryUser({ path: { id } })),
    onSuccess: async (host) => {
      toast.add({ title: "Primary user cleared", type: "success" });
      queryClient.setQueryData(hostKeys.detail(host.id), host);
      await queryClient.invalidateQueries({ queryKey: hostKeys.all });
    },
  });
}

export function useHostSoftware(id: number | null, params: HostSoftwareListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    source: params.source && params.source.length > 0 ? params.source : undefined,
  };

  return useQuery<PageHostSoftware, ApiError>({
    queryKey: hostKeys.software(id, queryParams),
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
    refetchInterval: HOST_REFRESH_MS,
  });
}

export function useHostOsqueryReports(id: number | null, params: HostOsqueryReportsParams = {}) {
  const queryParams = hostOsqueryReportsQueryParams(params);
  return useQuery<PageReportSnapshot, ApiError>({
    queryKey: hostKeys.osqueryReports(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(listHostOsqueryReports({ path: detailPath(id), query: queryParams, signal })),
    enabled: id !== null,
    placeholderData: keepPreviousData,
    refetchInterval: HOST_REFRESH_MS,
  });
}

export function useHostOsqueryChecks(id: number | null, params: HostOsqueryChecksParams = {}) {
  const queryParams = hostOsqueryChecksQueryParams(params);
  return useQuery<PageCheckHostStatus, ApiError>({
    queryKey: hostKeys.osqueryChecks(id, queryParams),
    queryFn: ({ signal }) =>
      unwrap(listHostOsqueryChecks({ path: detailPath(id), query: queryParams, signal })),
    enabled: id !== null,
    placeholderData: keepPreviousData,
    refetchInterval: HOST_REFRESH_MS,
  });
}

export function listAllHostOsqueryReports(
  id: number,
  params: HostOsqueryReportsParams = {},
): Promise<OsqueryReportSnapshot[]> {
  return collectAllPages((page, perPage) =>
    unwrap(
      listHostOsqueryReports({
        path: { id },
        query: hostOsqueryReportsQueryParams({ ...params, page, per_page: perPage }),
      }),
    ),
  );
}

function hostOsqueryReportsQueryParams(params: HostOsqueryReportsParams) {
  return {
    ...baseListParams(params),
    status: params.status,
  };
}

function hostOsqueryChecksQueryParams(params: HostOsqueryChecksParams) {
  return {
    ...baseListParams(params),
    status: params.status,
  };
}

export function useHostSantaRules(id: number | null, params: HostSantaRulesParams = {}) {
  const queryParams = baseListParams(params, {
    defaultPerPage: HOST_SANTA_RULES_PAGE_SIZE,
  });

  return useQuery<PageRuleStatus, ApiError>({
    queryKey: hostKeys.santaRules(id, queryParams),
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
    refetchInterval: HOST_REFRESH_MS,
  });
}
