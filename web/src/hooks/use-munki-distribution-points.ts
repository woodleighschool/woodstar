import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type {
  ApiError,
  MunkiDistributionPointDetail,
  MunkiDistributionPointKeyBody,
  MunkiDistributionPointMutation,
  MunkiRevealedDistributionPoint,
  PageDistributionPoint,
} from "@/lib/api";
import {
  createMunkiDistributionPoint,
  deleteMunkiDistributionPoint,
  listMunkiDistributionPoints,
  reorderMunkiDistributionPoints,
  rotateMunkiDistributionPointKey,
  unwrap,
  updateMunkiDistributionPoint,
} from "@/lib/api";
import type { ListMunkiDistributionPointsData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import {
  munkiDistributionPointQueryOptions,
  type MunkiDistributionPointRefreshOptions,
} from "@/lib/queries/munki";
import { queryKeys } from "@/lib/query-keys";

export type MunkiDistributionPointListParams = NonNullable<
  ListMunkiDistributionPointsData["query"]
>;

const MUNKI_DISTRIBUTION_DETAIL_REFRESH_MS = 5_000;
const MUNKI_DISTRIBUTION_LIST_REFRESH_MS = 30_000;

export function useMunkiDistributionPoints(params: MunkiDistributionPointListParams = {}) {
  const queryParams = baseListParams(params);

  return useQuery<PageDistributionPoint, ApiError>({
    queryKey: queryKeys.munkiDistributionPoints(queryParams),
    queryFn: ({ signal }) => unwrap(listMunkiDistributionPoints({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: MUNKI_DISTRIBUTION_LIST_REFRESH_MS,
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
    staleTime: MUNKI_DISTRIBUTION_DETAIL_REFRESH_MS,
    refetchInterval: MUNKI_DISTRIBUTION_DETAIL_REFRESH_MS,
    refetchIntervalInBackground: false,
  });
}

export function useCreateMunkiDistributionPoint() {
  const queryClient = useQueryClient();
  return useMutation<MunkiRevealedDistributionPoint, ApiError, MunkiDistributionPointMutation>({
    mutationFn: (body) => unwrap(createMunkiDistributionPoint({ body })),
    onSuccess: async (point) => {
      toast.success("Distribution point created");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.munkiDistributionPoint(point.id),
        }),
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
    mutationFn: ({ id, body }) =>
      unwrap(
        updateMunkiDistributionPoint({
          path: { id },
          body,
        }),
      ),
    onSuccess: async (point) => {
      toast.success("Distribution point saved");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.munkiDistributionPoint(point.id),
        }),
      ]);
    },
  });
}

export function useDeleteMunkiDistributionPoint() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) =>
      unwrap(
        deleteMunkiDistributionPoint({
          path: { id },
        }),
      ),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll });
    },
  });
}

export function useReorderMunkiDistributionPoints() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ordered_ids) => unwrap(reorderMunkiDistributionPoints({ body: { ordered_ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll });
    },
  });
}

export function useRotateMunkiDistributionPointKey() {
  const queryClient = useQueryClient();
  return useMutation<MunkiDistributionPointKeyBody, ApiError, number>({
    mutationFn: (id) =>
      unwrap(
        rotateMunkiDistributionPointKey({
          path: { id },
        }),
      ),
    onSuccess: async (_key, id) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.munkiDistributionPoint(id),
        }),
      ]);
    },
  });
}
