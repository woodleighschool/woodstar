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
  PageConfiguration,
  SantaConfiguration,
  SantaConfigurationMutation,
} from "@lib/api";
import {
  bulkDeleteSantaConfigurations,
  createSantaConfiguration,
  deleteSantaConfiguration,
  getSantaConfiguration,
  listSantaConfigurations,
  reorderSantaConfigurations,
  unwrap,
  updateSantaConfiguration,
} from "@lib/api";
import type { ListSantaConfigurationsData } from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

type QueryParams = Record<string, unknown>;

type SantaListParams = NonNullable<ListSantaConfigurationsData["query"]>;

const configurationKeys = {
  all: ["santa", "configurations"] as const,
  list: (params?: QueryParams) => ["santa", "configurations", "list", params ?? {}] as const,
  detail: (id: number | null) => ["santa", "configurations", "detail", id] as const,
};

export function santaConfigurationQueryOptions(id: number | null) {
  return queryOptions<SantaConfiguration, ApiError>({
    queryKey: configurationKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getSantaConfiguration({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function useSantaConfigurations(params: SantaListParams = {}) {
  const queryParams = baseListParams(params);

  return useQuery<PageConfiguration, ApiError>({
    queryKey: configurationKeys.list(queryParams),
    queryFn: ({ signal }) => unwrap(listSantaConfigurations({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaConfiguration(id: number | null) {
  return useQuery(santaConfigurationQueryOptions(id));
}

export function useCreateSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<SantaConfiguration, ApiError, SantaConfigurationMutation>({
    mutationFn: (body) => unwrap(createSantaConfiguration({ body })),
    onSuccess: async () => {
      toast.add({ title: "Configuration Created", type: "success" });
      await queryClient.invalidateQueries({ queryKey: configurationKeys.all });
    },
  });
}

export function useUpdateSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<
    SantaConfiguration,
    ApiError,
    { id: number; body: SantaConfigurationMutation }
  >({
    mutationFn: ({ id, body }) =>
      unwrap(
        updateSantaConfiguration({
          path: { id },
          body,
        }),
      ),
    onSuccess: async () => {
      toast.add({ title: "Configuration Saved", type: "success" });
      await queryClient.invalidateQueries({ queryKey: configurationKeys.all });
    },
  });
}

export function useDeleteSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteSantaConfiguration({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: configurationKeys.all });
    },
  });
}

export function useBulkDeleteSantaConfigurations() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteSantaConfigurations({ query: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: configurationKeys.all });
    },
  });
}

export function useReorderSantaConfigurations() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ordered_ids) => unwrap(reorderSantaConfigurations({ body: { ordered_ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: configurationKeys.all });
    },
  });
}
