import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ApiError, Department, Page, User, UserCreate, UserMutation } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import type { ListUserDepartmentsData, ListUsersData } from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { Department, User, UserCreate, UserMutation };
export type UserListResult = Page<User>;
export type DepartmentListResult = Page<Department>;
export type UserListParams = NonNullable<ListUsersData["query"]>;
export type DepartmentListParams = NonNullable<ListUserDepartmentsData["query"]>;

type BaseUserListParams = {
  page_index?: number;
  page_size?: number;
  q?: string;
  sort?: string;
  values?: string[] | null;
};

function baseUserQueryParams(params: BaseUserListParams = {}) {
  return {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
    values: params.values && params.values.length > 0 ? params.values : undefined,
  };
}

function userQueryParams(params: UserListParams = {}) {
  return {
    ...baseUserQueryParams(params),
    role: params.role,
    source: params.source,
    status: params.status,
    group_id: params.group_id,
  };
}

export function useUsers(params: UserListParams = {}) {
  const queryParams = userQueryParams(params);
  return useQuery<UserListResult, ApiError>({
    queryKey: queryKeys.users(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/users", {
          params: { query: queryParams },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useUserDepartments(params: DepartmentListParams = {}) {
  const queryParams = baseUserQueryParams(params);
  return useQuery<DepartmentListResult, ApiError>({
    queryKey: queryKeys.userDepartments(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/users/departments", {
          params: { query: queryParams },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
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
      await queryClient.invalidateQueries({ queryKey: ["users"] });
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
        queryClient.invalidateQueries({ queryKey: ["users"] }),
        queryClient.invalidateQueries({ queryKey: ["groups"] }),
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
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["users"] }),
        queryClient.invalidateQueries({ queryKey: ["groups"] }),
      ]);
    },
  });
}
