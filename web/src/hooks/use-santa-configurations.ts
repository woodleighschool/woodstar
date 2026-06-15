import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type {
  ApiError,
  Page,
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
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { SantaConfiguration, SantaConfigurationMutation };
export type SantaConfigurationListResult = Page<SantaConfiguration>;
export type SantaClientMode =
  | SantaHostState["client_mode_reported"]
  | SantaConfiguration["client_mode"];

export type SantaListParams = NonNullable<ListSantaConfigurationsData["query"]>;

export function useSantaConfigurations(params: SantaListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };

  return useQuery<SantaConfigurationListResult, ApiError>({
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
          path: { id: id ?? 0 },
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
    onSuccess: (configuration) => {
      toast.success("Configuration created");
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
      void queryClient.invalidateQueries({
        queryKey: queryKeys.santaConfiguration(configuration.id),
      });
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
    onSuccess: (configuration) => {
      toast.success("Configuration saved");
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
      void queryClient.invalidateQueries({
        queryKey: queryKeys.santaConfiguration(configuration.id),
      });
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
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
    },
  });
}

export function useBulkDeleteSantaConfigurations() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteSantaConfigurations({ body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
    },
  });
}

export function useReorderSantaConfigurations() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ordered_ids) => unwrap(reorderSantaConfigurations({ body: { ordered_ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
    },
  });
}
