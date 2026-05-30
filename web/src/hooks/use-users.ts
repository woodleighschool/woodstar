import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError, Page, User, UserCreate, UserMutation } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type { User, UserCreate, UserMutation };
export type UserListResult = Page<User>;

export function useUsers() {
  return useQuery<User[], ApiError>({
    queryKey: queryKeys.users,
    queryFn: async ({ signal }) => {
      const result = await unwrap(apiClient.GET<UserListResult>("/api/users", { signal }));
      return result.items ?? [];
    },
  });
}

export function useUser(id: number | null) {
  return useQuery<User, ApiError>({
    queryKey: queryKeys.user(id),
    queryFn: async ({ signal }) =>
      unwrap(
        apiClient.GET("/api/users/{id}", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();
  return useMutation<User, ApiError, UserCreate>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/users", { body })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.users });
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  return useMutation<User, ApiError, { id: number; body: UserMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(
        apiClient.PUT("/api/users/{id}", {
          params: { path: { id } },
          body,
        }),
      ),
    onSuccess: async (user, variables) => {
      queryClient.setQueryData(queryKeys.user(variables.id), user);
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
          params: { path: { id } },
        }),
      );
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.users });
    },
  });
}
