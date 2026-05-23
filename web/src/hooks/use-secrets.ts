import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type EnrollSecret = Schemas["EnrollSecret"];

export function useEnrollSecrets() {
  return useQuery<EnrollSecret[], ApiError>({
    queryKey: queryKeys.enrollSecrets,
    queryFn: async ({ signal }) => (await unwrap(apiClient.GET("/api/enroll-secrets", { signal }))) ?? [],
  });
}

export function useCreateEnrollSecret() {
  const queryClient = useQueryClient();
  return useMutation<EnrollSecret, ApiError>({
    mutationFn: () => unwrap(apiClient.POST("/api/enroll-secrets")),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.enrollSecrets });
    },
  });
}

export function useDeleteEnrollSecret() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: async (id) => {
      await unwrap(
        apiClient.DELETE("/api/enroll-secrets/{id}", {
          params: { path: { id: String(id) } },
        }),
      );
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.enrollSecrets });
    },
  });
}
