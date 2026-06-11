import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type { ApiError, Configuration, ConfigurationMutation, Page, SantaHostState } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import type { ListSantaConfigurationsData } from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type SantaConfiguration = Configuration;
export type SantaConfigurationMutation = ConfigurationMutation;
export type SantaConfigurationListResult = Page<SantaConfiguration>;
export type SantaClientMode = SantaHostState["client_mode_reported"] | SantaConfiguration["client_mode"];

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
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/santa/configurations", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaConfiguration(id: number | null) {
  return useQuery<SantaConfiguration, ApiError>({
    queryKey: queryKeys.santaConfiguration(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/santa/configurations/{id}", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<SantaConfiguration, ApiError, SantaConfigurationMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/santa/configurations", { body })),
    meta: { inlineError: true },
    onSuccess: (configuration) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfiguration(configuration.id) });
    },
  });
}

export function useUpdateSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<SantaConfiguration, ApiError, { id: number; body: SantaConfigurationMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(
        apiClient.PUT("/api/santa/configurations/{id}", {
          params: { path: { id } },
          body,
        }),
      ),
    meta: { inlineError: true },
    onSuccess: (configuration) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfiguration(configuration.id) });
    },
  });
}

export function useDeleteSantaConfiguration() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) =>
      unwrap(
        apiClient.DELETE("/api/santa/configurations/{id}", {
          params: { path: { id } },
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
    mutationFn: (ids) => unwrap(apiClient.POST("/api/santa/configurations/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
    },
  });
}

export function useReorderSantaConfigurations() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ordered_ids) => unwrap(apiClient.PUT("/api/santa/configurations/order", { body: { ordered_ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.santaConfigurationsAll });
    },
  });
}
