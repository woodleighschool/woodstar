import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type Secret = Schemas["Secret"];

export function useEnrollSecrets() {
  return useQuery<Secret[], ApiError>({
    queryKey: queryKeys.enrollSecrets,
    queryFn: async ({ signal }) => (await unwrap(apiClient.GET("/api/orbit/enroll-secrets", { signal }))) ?? [],
  });
}

export function useCreateEnrollSecret() {
  const queryClient = useQueryClient();
  return useMutation<Secret, ApiError>({
    mutationFn: () => unwrap(apiClient.POST("/api/orbit/enroll-secrets")),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.enrollSecrets });
    },
  });
}

export function useDeleteEnrollSecret() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, string>({
    mutationFn: async (id) => {
      await unwrap(
        apiClient.DELETE("/api/orbit/enroll-secrets/{id}", {
          params: { path: { id } },
        }),
      );
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.enrollSecrets });
    },
  });
}
