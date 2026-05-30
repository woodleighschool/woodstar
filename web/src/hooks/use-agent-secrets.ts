import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { AgentSecret, AgentSecretCreate, AgentSecretMutation, ApiError } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type { AgentSecret, AgentSecretCreate, AgentSecretMutation };
export type Agent = AgentSecret["agent"];

export function useAgentSecrets() {
  return useQuery<AgentSecret[], ApiError>({
    queryKey: queryKeys.agentSecrets,
    queryFn: async ({ signal }) =>
      (await unwrap(apiClient.GET<AgentSecret[] | null>("/api/agent-secrets", { signal }))) ?? [],
  });
}

export function useCreateAgentSecret() {
  const queryClient = useQueryClient();
  return useMutation<AgentSecret, ApiError, AgentSecretCreate>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/agent-secrets", { body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.agentSecrets });
    },
  });
}

export function useUpdateAgentSecret() {
  const queryClient = useQueryClient();
  return useMutation<AgentSecret, ApiError, { id: number; body: AgentSecretMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(apiClient.PATCH("/api/agent-secrets/{id}", { params: { path: { id } }, body })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.agentSecrets });
    },
  });
}

export function useDeleteAgentSecret() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/agent-secrets/{id}", { params: { path: { id } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.agentSecrets });
    },
  });
}
