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
  MunkiDistributionPointDetail,
  MunkiDistributionPointKeyBody,
  MunkiDistributionPointMutation,
  MunkiRevealedDistributionPoint,
  PageDistributionPoint,
} from "@lib/api";
import {
  createMunkiDistributionPoint,
  deleteMunkiDistributionPoint,
  getMunkiDistributionPoint,
  listMunkiDistributionPoints,
  reorderMunkiDistributionPoints,
  rotateMunkiDistributionPointKey,
  unwrap,
  updateMunkiDistributionPoint,
} from "@lib/api";
import type { ListMunkiDistributionPointsData } from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

type MunkiDistributionPointListParams = NonNullable<ListMunkiDistributionPointsData["query"]>;

interface MunkiDistributionPointRefreshOptions {
  staleTime?: number;
  refetchInterval?: number | false;
  refetchIntervalInBackground?: boolean;
}

type QueryParams = Record<string, unknown>;

const distributionPointRoot = ["munki", "distribution-points"] as const;
const detailRefreshMs = 5_000;
const listRefreshMs = 30_000;

const munkiDistributionPointKeys = {
  root: distributionPointRoot,
  list: (params: QueryParams) => [...distributionPointRoot, "list", params] as const,
  detail: (id: number | null) => [...distributionPointRoot, "detail", id] as const,
};

export function munkiDistributionPointQueryOptions(
  id: number | null,
  refreshOptions: MunkiDistributionPointRefreshOptions = {},
) {
  return queryOptions<MunkiDistributionPointDetail, ApiError>({
    queryKey: munkiDistributionPointKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getMunkiDistributionPoint({ path: detailPath(id), signal })),
    enabled: id !== null,
    ...refreshOptions,
  });
}

export function useMunkiDistributionPoints(params: MunkiDistributionPointListParams = {}) {
  const query = baseListParams(params);
  return useQuery<PageDistributionPoint, ApiError>({
    queryKey: munkiDistributionPointKeys.list(query),
    queryFn: ({ signal }) => unwrap(listMunkiDistributionPoints({ query, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: listRefreshMs,
  });
}

export function useMunkiDistributionPoint(
  id: number | null,
  refreshOptions: MunkiDistributionPointRefreshOptions = {},
) {
  return useQuery(munkiDistributionPointQueryOptions(id, refreshOptions));
}

export function useLiveMunkiDistributionPoint(id: number | null) {
  return useMunkiDistributionPoint(id, {
    staleTime: detailRefreshMs,
    refetchInterval: detailRefreshMs,
    refetchIntervalInBackground: false,
  });
}

export function useCreateMunkiDistributionPoint() {
  const queryClient = useQueryClient();
  return useMutation<MunkiRevealedDistributionPoint, ApiError, MunkiDistributionPointMutation>({
    mutationFn: (body) => unwrap(createMunkiDistributionPoint({ body })),
    onSuccess: async (point) => {
      toast.add({ title: "Distribution Point Created", type: "success" });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: munkiDistributionPointKeys.root }),
        queryClient.invalidateQueries({ queryKey: munkiDistributionPointKeys.detail(point.id) }),
      ]);
    },
  });
}

export function useUpdateMunkiDistributionPoint() {
  const queryClient = useQueryClient();
  return useMutation<
    MunkiDistributionPointDetail,
    ApiError,
    { id: number; body: MunkiDistributionPointMutation }
  >({
    mutationFn: ({ id, body }) => unwrap(updateMunkiDistributionPoint({ path: { id }, body })),
    onSuccess: async (point) => {
      toast.add({ title: "Distribution Point Saved", type: "success" });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: munkiDistributionPointKeys.root }),
        queryClient.invalidateQueries({ queryKey: munkiDistributionPointKeys.detail(point.id) }),
      ]);
    },
  });
}

export function useDeleteMunkiDistributionPoint() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteMunkiDistributionPoint({ path: { id } })),
    onSuccess: async () =>
      queryClient.invalidateQueries({ queryKey: munkiDistributionPointKeys.root }),
  });
}

export function useReorderMunkiDistributionPoints() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ordered_ids) => unwrap(reorderMunkiDistributionPoints({ body: { ordered_ids } })),
    onSuccess: async () =>
      queryClient.invalidateQueries({ queryKey: munkiDistributionPointKeys.root }),
  });
}

export function useRotateMunkiDistributionPointKey() {
  const queryClient = useQueryClient();
  return useMutation<MunkiDistributionPointKeyBody, ApiError, number>({
    mutationFn: (id) => unwrap(rotateMunkiDistributionPointKey({ path: { id } })),
    onSuccess: async (_key, id) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: munkiDistributionPointKeys.root }),
        queryClient.invalidateQueries({ queryKey: munkiDistributionPointKeys.detail(id) }),
      ]);
    },
  });
}
