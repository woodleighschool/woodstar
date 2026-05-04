import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { ApiError, apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type User = Schemas["UserBody"];
export type UserCreateBody = Schemas["UserCreateInputBody"];
export type UserUpdateBody = Schemas["UserUpdateInputBody"];

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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users });
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  return useMutation<User, ApiError, { id: string; body: UserUpdateBody }>({
    mutationFn: ({ id, body }) =>
      unwrap(
        apiClient.PATCH("/api/users/{id}", {
          params: { path: { id } },
          body,
        }),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users });
      // The actor may have updated their own row (name/password).
      queryClient.invalidateQueries({ queryKey: queryKeys.authMe });
    },
  });
}

export function useDeleteUser() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, string>({
    mutationFn: async (id) => {
      await unwrap(
        apiClient.DELETE("/api/users/{id}", {
          params: { path: { id } },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users });
    },
  });
}
