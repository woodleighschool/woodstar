import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type {
  ApiError,
  MunkiCreateMutation,
  MunkiSoftwareDetail,
  MunkiUpdateMutation,
  PageMunkiSoftware,
} from "@/lib/api";
import {
  bulkDeleteMunkiSoftware,
  createMunkiSoftware,
  deleteMunkiSoftware,
  getMunkiSoftware,
  listMunkiSoftware,
  unwrap,
  updateMunkiSoftware,
} from "@/lib/api";
import type { ListMunkiSoftwareData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

type MunkiListParams = NonNullable<ListMunkiSoftwareData["query"]>;

export function useMunkiSoftware(params: MunkiListParams = {}) {
  const query = baseListParams(params);
  return useQuery<PageMunkiSoftware, ApiError>({
    queryKey: queryKeys.munkiSoftware(query),
    queryFn: ({ signal }) => unwrap(listMunkiSoftware({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiSoftwareDetail(id: number | null) {
  return useQuery<MunkiSoftwareDetail, ApiError>({
    queryKey: queryKeys.munkiSoftwareDetail(id),
    queryFn: ({ signal }) => unwrap(getMunkiSoftware({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
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
    onSuccess: async (title) => {
      toast.success("Software saved");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(title.id) }),
      ]);
    },
  });
}

export function useDeleteMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteMunkiSoftware({ path: { id } })),
    onSuccess: async (_data, id) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(id) }),
      ]);
    },
  });
}

export function useBulkDeleteMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteMunkiSoftware({ body: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
    },
  });
}
