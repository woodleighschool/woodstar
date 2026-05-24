import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type AgentSecret = Schemas["AgentSecret"];
export type Agent = AgentSecret["agent"];

export function useAgentSecrets() {
  return useQuery<AgentSecret[], ApiError>({
    queryKey: queryKeys.agentSecrets,
    queryFn: async ({ signal }) => (await unwrap(apiClient.GET("/api/agent-secrets", { signal }))) ?? [],
  });
}

export function useCreateAgentSecret() {
  const queryClient = useQueryClient();
  return useMutation<AgentSecret, ApiError, { agent: Agent; value: string }>({
    mutationFn: ({ agent, value }) => unwrap(apiClient.POST("/api/agent-secrets", { body: { agent, value } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.agentSecrets });
    },
  });
}

export function useUpdateAgentSecret() {
  const queryClient = useQueryClient();
  return useMutation<AgentSecret, ApiError, { id: number; value: string }>({
    mutationFn: ({ id, value }) =>
      unwrap(apiClient.PATCH("/api/agent-secrets/{id}", { params: { path: { id: String(id) } }, body: { value } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.agentSecrets });
    },
  });
}

export function useDeleteAgentSecret() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(apiClient.DELETE("/api/agent-secrets/{id}", { params: { path: { id: String(id) } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.agentSecrets });
    },
  });
}
