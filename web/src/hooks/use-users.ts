import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import type {
  ApiError,
  Department,
  PageDepartment,
  PageUser,
  User,
  UserCreate,
  UserMutation,
} from "@/lib/api";
import {
  createUser,
  deleteUser,
  getUser,
  listUserDepartments,
  listUsers,
  unwrap,
  updateUser,
} from "@/lib/api";
import type { ListUserDepartmentsData, ListUsersData } from "@/lib/api-client/types.gen";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { Department, User, UserCreate, UserMutation };
export type UserListResult = PageUser;
export type DepartmentListResult = PageDepartment;
export type UserListParams = NonNullable<ListUsersData["query"]>;
export type DepartmentListParams = NonNullable<ListUserDepartmentsData["query"]>;

type BaseUserListParams = {
  page?: number;
  per_page?: number;
  q?: string;
  sort?: string;
  values?: string[] | null;
};

function baseUserQueryParams(params: BaseUserListParams = {}) {
  return {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
    values: params.values && params.values.length > 0 ? params.values : undefined,
  };
}

function userQueryParams(params: UserListParams = {}) {
  return {
    ...baseUserQueryParams(params),
    role: params.role,
    source: params.source,
    group_id: params.group_id,
  };
}

export function useUsers(params: UserListParams = {}) {
  const queryParams = userQueryParams(params);
  return useQuery<UserListResult, ApiError>({
    queryKey: queryKeys.users(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listUsers({
          query: queryParams,
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
        listUserDepartments({
          query: queryParams,
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useUser(id: number | null) {
  return useQuery<User, ApiError>({
    queryKey: queryKeys.user(id),
    enabled: id !== null,
    queryFn: async ({ signal }) =>
      unwrap(
        getUser({
          path: { id: id ?? 0 },
          signal,
        }),
      ),
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();
  return useMutation<User, ApiError, UserCreate>({
    mutationFn: (body) => unwrap(createUser({ body })),
    onSuccess: async () => {
      toast.success("User created");
      await queryClient.invalidateQueries({ queryKey: queryKeys.usersAll });
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  return useMutation<User, ApiError, { id: number; body: UserMutation }>({
    mutationFn: ({ id, body }) =>
      unwrap(
        updateUser({
          path: { id },
          body,
        }),
      ),
    onSuccess: async (user, variables) => {
      toast.success("User saved");
      queryClient.setQueryData(queryKeys.user(variables.id), user);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.usersAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.groupsAll }),
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
        deleteUser({
          path: { id },
        }),
      );
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.usersAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.groupsAll }),
      ]);
    },
  });
}
