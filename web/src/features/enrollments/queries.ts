import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { AgentSecret, AgentSecretCreate, AgentSecretMutation, ApiError } from "@lib/api";
import {
  createAgentSecret,
  deleteAgentSecret,
  listAgentSecrets,
  unwrap,
  updateAgentSecret,
} from "@lib/api";

const agentSecretsKey = ["agent-secrets"] as const;

export function useAgentSecrets(enabled = true) {
  return useQuery<AgentSecret[], ApiError>({
    queryKey: agentSecretsKey,
    queryFn: ({ signal }) => unwrap(listAgentSecrets({ signal })),
    enabled,
  });
}

export function useCreateAgentSecret() {
  const queryClient = useQueryClient();
  return useMutation<AgentSecret, ApiError, AgentSecretCreate>({
    mutationFn: (body) => unwrap(createAgentSecret({ body })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: agentSecretsKey });
    },
  });
}

export function useUpdateAgentSecret() {
  const queryClient = useQueryClient();
  return useMutation<AgentSecret, ApiError, { id: number; body: AgentSecretMutation }>({
    mutationFn: ({ id, body }) => unwrap(updateAgentSecret({ path: { id }, body })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: agentSecretsKey });
    },
  });
}

export function useDeleteAgentSecret() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteAgentSecret({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: agentSecretsKey });
    },
  });
}
