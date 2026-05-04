import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { ApiError, apiClient, unwrap, type Schemas } from "@/lib/api";
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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.enrollSecrets });
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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.enrollSecrets });
    },
  });
}

export function useSantaTokens() {
  return useQuery<Secret[], ApiError>({
    queryKey: queryKeys.santaTokens,
    queryFn: async ({ signal }) => (await unwrap(apiClient.GET("/api/santa/tokens", { signal }))) ?? [],
  });
}

export function useCreateSantaToken() {
  const queryClient = useQueryClient();
  return useMutation<Secret, ApiError>({
    mutationFn: () => unwrap(apiClient.POST("/api/santa/tokens")),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.santaTokens });
    },
  });
}

export function useDeleteSantaToken() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, string>({
    mutationFn: async (id) => {
      await unwrap(
        apiClient.DELETE("/api/santa/tokens/{id}", {
          params: { path: { id } },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.santaTokens });
    },
  });
}

export function useMunkiTokens() {
  return useQuery<Secret[], ApiError>({
    queryKey: queryKeys.munkiTokens,
    queryFn: async ({ signal }) => (await unwrap(apiClient.GET("/api/munki/tokens", { signal }))) ?? [],
  });
}

export function useCreateMunkiToken() {
  const queryClient = useQueryClient();
  return useMutation<Secret, ApiError>({
    mutationFn: () => unwrap(apiClient.POST("/api/munki/tokens")),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.munkiTokens });
    },
  });
}

export function useDeleteMunkiToken() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, string>({
    mutationFn: async (id) => {
      await unwrap(
        apiClient.DELETE("/api/munki/tokens/{id}", {
          params: { path: { id } },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.munkiTokens });
    },
  });
}
