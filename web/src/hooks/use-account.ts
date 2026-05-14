import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type Account = Schemas["AccountBody"];

export function useAccount() {
  return useQuery<Account, ApiError>({
    queryKey: queryKeys.account,
    queryFn: async ({ signal }) => unwrap(apiClient.GET("/api/account", { signal })),
    staleTime: 30_000,
  });
}

export function useRotateAPIKey() {
  const queryClient = useQueryClient();
  return useMutation<Account, ApiError>({
    mutationFn: () => unwrap(apiClient.POST("/api/account/api-key")),
    onSuccess: (account) => {
      queryClient.setQueryData(queryKeys.account, account);
    },
  });
}

export function useRevokeAPIKey() {
  const queryClient = useQueryClient();
  return useMutation<Account, ApiError>({
    mutationFn: () => unwrap(apiClient.DELETE("/api/account/api-key")),
    onSuccess: (account) => {
      queryClient.setQueryData(queryKeys.account, account);
    },
  });
}
