import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { Account, AccountMutation, ApiError, Session } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type { Account, AccountMutation };

export function useAccount() {
  return useQuery<Account, ApiError>({
    queryKey: queryKeys.account,
    queryFn: async ({ signal }) => unwrap(apiClient.GET("/api/account", { signal })),
    staleTime: 30_000,
  });
}

export function useUpdateAccount() {
  const queryClient = useQueryClient();
  return useMutation<Account, ApiError, AccountMutation>({
    mutationFn: (body) => unwrap(apiClient.PUT("/api/account", { body })),
    onSuccess: async (account) => {
      queryClient.setQueryData(queryKeys.account, account);
      queryClient.setQueryData(queryKeys.user(account.user.id), account.user);
      queryClient.setQueryData(queryKeys.session, (session: Session | undefined) =>
        session ? { ...session, user: account.user } : session,
      );
      await queryClient.invalidateQueries({ queryKey: ["users"] });
    },
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
