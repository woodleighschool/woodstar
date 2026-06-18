import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type {
  ApiError,
  MunkiDistributionPoint,
  MunkiDistributionPointDetail,
  MunkiDistributionPointKeyBody,
  MunkiDistributionPointMutation,
  MunkiRevealedDistributionPoint,
  Page,
} from "@/lib/api";
import {
  createMunkiDistributionPoint,
  deleteMunkiDistributionPoint,
  getMunkiDistributionPoint,
  listMunkiDistributionPoints,
  reorderMunkiDistributionPoints,
  rotateMunkiDistributionPointKey,
  unwrap,
  updateMunkiDistributionPoint,
} from "@/lib/api";
import type { ListMunkiDistributionPointsData } from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type {
  MunkiDistributionPoint,
  MunkiDistributionPointDetail,
  MunkiDistributionPointMutation,
};
export type MunkiDistributionPointListResult = Page<MunkiDistributionPoint>;

export type MunkiDistributionPointListParams = NonNullable<
  ListMunkiDistributionPointsData["query"]
>;

type MunkiDistributionPointDetailRefreshOptions = {
  staleTime?: number;
  refetchInterval?: number | false;
  refetchIntervalInBackground?: boolean;
};

const MUNKI_DISTRIBUTION_DETAIL_REFRESH_MS = 5_000;

export function useMunkiDistributionPoints(params: MunkiDistributionPointListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };

  return useQuery<MunkiDistributionPointListResult, ApiError>({
    queryKey: queryKeys.munkiDistributionPoints(queryParams),
    queryFn: ({ signal }) => unwrap(listMunkiDistributionPoints({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiDistributionPoint(
  id: number | null,
  refreshOptions: MunkiDistributionPointDetailRefreshOptions = {},
) {
  return useQuery<MunkiDistributionPointDetail, ApiError>({
    queryKey: queryKeys.munkiDistributionPoint(id),
    queryFn: ({ signal }) =>
      unwrap(
        getMunkiDistributionPoint({
          path: { id: id ?? 0 },
          signal,
        }),
      ),
    enabled: id !== null,
    ...refreshOptions,
  });
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
    onSuccess: (point) => {
      toast.success("Distribution point created");
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll });
      void queryClient.invalidateQueries({
        queryKey: queryKeys.munkiDistributionPoint(point.id),
      });
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
    onSuccess: (point) => {
      toast.success("Distribution point saved");
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll });
      void queryClient.invalidateQueries({
        queryKey: queryKeys.munkiDistributionPoint(point.id),
      });
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
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll });
    },
  });
}

export function useReorderMunkiDistributionPoints() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ordered_ids) => unwrap(reorderMunkiDistributionPoints({ body: { ordered_ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll });
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
    onSuccess: (_key, id) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiDistributionPointsAll });
      void queryClient.invalidateQueries({
        queryKey: queryKeys.munkiDistributionPoint(id),
      });
    },
  });
}
