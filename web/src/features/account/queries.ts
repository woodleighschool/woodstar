import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { accountKey, accountQueryOptions } from "@features/account/query-options";
import { sessionQueryOptions } from "@features/authn/queries";
import { userKeys } from "@features/directory/users/queries";
import type { Account, AccountMutation, ApiError, SessionBody } from "@lib/api";
import { revokeAccountApiKey, rotateAccountApiKey, unwrap, updateAccount } from "@lib/api";
export function useAccount() {
  return useQuery(accountQueryOptions);
}

export function useUpdateAccount() {
  const queryClient = useQueryClient();
  return useMutation<Account, ApiError, AccountMutation>({
    mutationFn: (body) => unwrap(updateAccount({ body })),
    onSuccess: async (account) => {
      queryClient.setQueryData(accountKey, account);
      queryClient.setQueryData(userKeys.detail(account.user.id), account.user);
      queryClient.setQueryData(sessionQueryOptions.queryKey, (session: SessionBody | undefined) => {
        if (!session) return session;
        return { ...session, user: account.user };
      });
      await queryClient.invalidateQueries({ queryKey: userKeys.all });
    },
  });
}

export function useRotateAPIKey() {
  const queryClient = useQueryClient();
  return useMutation<Account, ApiError>({
    mutationFn: () => unwrap(rotateAccountApiKey()),
    onSuccess: (account) => {
      queryClient.setQueryData(accountKey, account);
    },
  });
}

export function useRevokeAPIKey() {
  const queryClient = useQueryClient();
  return useMutation<Account, ApiError>({
    mutationFn: () => unwrap(revokeAccountApiKey()),
    onSuccess: (account) => {
      queryClient.setQueryData(accountKey, account);
    },
  });
}
