import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type {
  ApiError,
  MunkiMutation,
  MunkiSoftware,
  MunkiSoftwareDetail,
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
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { MunkiSoftware, MunkiSoftwareDetail, MunkiMutation };

type MunkiListParams = NonNullable<ListMunkiSoftwareData["query"]>;

function queryParams(params: MunkiListParams) {
  return {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };
}

export function useMunkiSoftware(params: MunkiListParams = {}) {
  const query = queryParams(params);
  return useQuery<PageMunkiSoftware, ApiError>({
    queryKey: queryKeys.munkiSoftware(query),
    queryFn: ({ signal }) => unwrap(listMunkiSoftware({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiSoftwareDetail(id: number | null) {
  return useQuery<MunkiSoftwareDetail, ApiError>({
    queryKey: queryKeys.munkiSoftwareDetail(id),
    queryFn: ({ signal }) => unwrap(getMunkiSoftware({ path: { id: id ?? 0 }, signal })),
    enabled: id !== null,
  });
}

export function useCreateMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareDetail, ApiError, MunkiMutation>({
    mutationFn: (body) => unwrap(createMunkiSoftware({ body })),
    onSuccess: () => {
      toast.success("Software created");
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
    },
  });
}

export function useUpdateMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareDetail, ApiError, { id: number; body: MunkiMutation }>({
    mutationFn: ({ id, body }) => unwrap(updateMunkiSoftware({ path: { id }, body })),
    onSuccess: (title) => {
      toast.success("Software saved");
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(title.id) });
    },
  });
}

export function useDeleteMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteMunkiSoftware({ path: { id } })),
    onSuccess: (_data, id) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareDetail(id) });
    },
  });
}

export function useBulkDeleteMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteMunkiSoftware({ body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiSoftwareAll });
    },
  });
}
