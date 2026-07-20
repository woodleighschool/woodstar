import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type {
  ApiError,
  MunkiCreateMutation,
  MunkiSoftwareDetail,
  MunkiUpdateMutation,
  PageSoftware,
} from "@/lib/api";
import {
  bulkDeleteMunkiSoftware,
  createMunkiSoftware,
  deleteMunkiSoftware,
  listMunkiSoftware,
  unwrap,
  updateMunkiSoftware,
} from "@/lib/api";
import type { ListMunkiSoftwareData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { invalidateMunkiSoftwareProjections, munkiSoftwareQueryOptions } from "@/lib/queries/munki";
import { queryKeys } from "@/lib/query-keys";

type MunkiListParams = NonNullable<ListMunkiSoftwareData["query"]>;

export function useMunkiSoftware(params: MunkiListParams = {}) {
  const query = baseListParams(params);
  return useQuery<PageSoftware, ApiError>({
    queryKey: queryKeys.munkiSoftware(query),
    queryFn: ({ signal }) => unwrap(listMunkiSoftware({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiSoftwareDetail(id: number | null) {
  return useQuery(munkiSoftwareQueryOptions(id));
}

export function useCreateMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareDetail, ApiError, MunkiCreateMutation>({
    mutationFn: (body) => unwrap(createMunkiSoftware({ body })),
    onSuccess: async () => {
      toast.success("Software created");
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
    },
  });
}

export function useUpdateMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareDetail, ApiError, { id: number; body: MunkiUpdateMutation }>({
    mutationFn: ({ id, body }) => unwrap(updateMunkiSoftware({ path: { id }, body })),
    onSuccess: async () => {
      toast.success("Software saved");
      await invalidateMunkiSoftwareProjections(queryClient);
    },
  });
}

export function useDeleteMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteMunkiSoftware({ path: { id } })),
    onSuccess: async () => {
      await invalidateMunkiSoftwareProjections(queryClient);
    },
  });
}

export function useBulkDeleteMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteMunkiSoftware({ query: { ids } })),
    onSuccess: async () => {
      await invalidateMunkiSoftwareProjections(queryClient);
    },
  });
}
