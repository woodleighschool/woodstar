import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type User = Schemas["User"];
export type UserCreateBody = Schemas["UserCreateInputBody"];
export type UserUpdateBody = Schemas["UserPutBody"];

export function useUsers() {
  return useQuery<User[], ApiError>({
    queryKey: queryKeys.users,
    queryFn: async ({ signal }) => (await unwrap(apiClient.GET("/api/users", { signal }))) ?? [],
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();
  return useMutation<User, ApiError, UserCreateBody>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/users", { body })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.users });
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  return useMutation<User, ApiError, { id: number; body: UserUpdateBody }>({
    mutationFn: ({ id, body }) =>
      unwrap(
        apiClient.PUT("/api/users/{id}", {
          params: { path: { id: String(id) } },
          body,
        }),
      ),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.users }),
        queryClient.invalidateQueries({ queryKey: queryKeys.session }),
      ]);
    },
  });
}

export function useDeleteUser() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: async (id) => {
      await unwrap(
        apiClient.DELETE("/api/users/{id}", {
          params: { path: { id: String(id) } },
        }),
      );
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.users });
    },
  });
}
