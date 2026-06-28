import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type {
  ApiError,
  PageConfiguration,
  SantaConfiguration,
  SantaConfigurationMutation,
  SantaHostState,
} from "@/lib/api";
import {
  bulkDeleteSantaConfigurations,
  createSantaConfiguration,
  deleteSantaConfiguration,
  getSantaConfiguration,
  listSantaConfigurations,
  reorderSantaConfigurations,
  unwrap,
  updateSantaConfiguration,
} from "@/lib/api";
import type { ListSantaConfigurationsData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export type SantaClientMode =
  | SantaHostState["client_mode_reported"]
  | SantaConfiguration["client_mode"];

export type SantaListParams = NonNullable<ListSantaConfigurationsData["query"]>;

export function useSantaConfigurations(params: SantaListParams = {}) {
  const queryParams = baseListParams(params);

  return useQuery<PageConfiguration, ApiError>({
    queryKey: queryKeys.santaConfigurations(queryParams),
    queryFn: ({ signal }) => unwrap(listSantaConfigurations({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaConfiguration(id: number | null) {
  return useQuery<SantaConfiguration, ApiError>({
    queryKey: queryKeys.santaConfiguration(id),
    queryFn: ({ signal }) =>
      unwrap(
        getSantaConfiguration({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<SantaConfiguration, ApiError, SantaConfigurationMutation>({
    mutationFn: (body) => unwrap(createSantaConfiguration({ body })),
    onSuccess: async (configuration) => {
      toast.success("Configuration created");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.santaConfiguration(configuration.id),
        }),
      ]);
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
    onSuccess: async (configuration) => {
      toast.success("Configuration saved");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.santaConfiguration(configuration.id),
        }),
      ]);
    },
  });
}

export function useDeleteSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) =>
      unwrap(
        deleteSantaConfiguration({
          path: { id },
        }),
      ),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
    },
  });
}

export function useBulkDeleteSantaConfigurations() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteSantaConfigurations({ body: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
    },
  });
}

export function useReorderSantaConfigurations() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ordered_ids) => unwrap(reorderSantaConfigurations({ body: { ordered_ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
    },
  });
}
