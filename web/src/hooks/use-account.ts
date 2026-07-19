import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { Account, AccountMutation, ApiError, SessionBody } from "@/lib/api";
import {
  getAccount,
  revokeAccountApiKey,
  rotateAccountApiKey,
  unwrap,
  updateAccount,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { sessionQueryOptions } from "@/lib/session";

export function useAccount() {
  return useQuery<Account, ApiError>({
    queryKey: queryKeys.account,
    queryFn: async ({ signal }) => unwrap(getAccount({ signal })),
  });
}

export function useUpdateAccount() {
  const queryClient = useQueryClient();
  return useMutation<Account, ApiError, AccountMutation>({
    mutationFn: (body) => unwrap(updateAccount({ body })),
    onSuccess: async (account) => {
      queryClient.setQueryData(queryKeys.account, account);
      queryClient.setQueryData(queryKeys.user(account.user.id), account.user);
      queryClient.setQueryData(sessionQueryOptions.queryKey, (session: SessionBody | undefined) => {
        if (!session || !account.user.role) return session;
        return {
          ...session,
          user: {
            id: account.user.id,
            email: account.user.email,
            name: account.user.name,
            role: account.user.role,
          },
        };
      });
      await queryClient.invalidateQueries({ queryKey: queryKeys.usersAll });
    },
  });
}

export function useRotateAPIKey() {
  const queryClient = useQueryClient();
  return useMutation<Account, ApiError>({
    mutationFn: () => unwrap(rotateAccountApiKey()),
    onSuccess: (account) => {
      queryClient.setQueryData(queryKeys.account, account);
    },
  });
}

export function useRevokeAPIKey() {
  const queryClient = useQueryClient();
  return useMutation<Account, ApiError>({
    mutationFn: () => unwrap(revokeAccountApiKey()),
    onSuccess: (account) => {
      queryClient.setQueryData(queryKeys.account, account);
    },
  });
}
