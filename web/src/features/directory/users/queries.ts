import {
  keepPreviousData,
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import { sessionQueryOptions } from "@features/authn/queries";
import { groupKeys } from "@features/directory/groups/queries";
import type { ApiError, PageDepartment, PageUser, User, UserCreate, UserMutation } from "@lib/api";
import {
  createUser,
  deleteUser,
  getUser,
  listUserDepartments,
  listUsers,
  unwrap,
  updateUser,
} from "@lib/api";
import type { ListUserDepartmentsData, ListUsersData } from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

export type UserListParams = NonNullable<ListUsersData["query"]>;
export type DepartmentListParams = NonNullable<ListUserDepartmentsData["query"]>;

type BaseUserListParams = {
  page?: number;
  per_page?: number;
  q?: string;
  sort?: string;
  values?: string[] | null;
};

type QueryParams = Record<string, unknown>;

export const userKeys = {
  all: ["users"] as const,
  list: (params?: QueryParams) => ["users", "list", params ?? {}] as const,
  detail: (id: number | null) => ["users", "detail", id] as const,
  departments: (params?: QueryParams) => ["users", "departments", "list", params ?? {}] as const,
};

function baseUserQueryParams(params: BaseUserListParams = {}) {
  return {
    ...baseListParams(params),
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
  return useQuery<PageUser, ApiError>({
    queryKey: userKeys.list(queryParams),
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
  return useQuery<PageDepartment, ApiError>({
    queryKey: userKeys.departments(queryParams),
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

export function useCreateUser() {
  const queryClient = useQueryClient();
  return useMutation<User, ApiError, UserCreate>({
    mutationFn: (body) => unwrap(createUser({ body })),
    onSuccess: async () => {
      toast.add({ title: "User Created", type: "success" });
      await queryClient.invalidateQueries({ queryKey: userKeys.all });
    },
  });
}

export function useUser(id: number | null) {
  return useQuery(userQueryOptions(id));
}

export function userQueryOptions(id: number | null) {
  return queryOptions<User, ApiError>({
    queryKey: userKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getUser({ path: detailPath(id), signal })),
    enabled: id !== null,
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
      toast.add({ title: "User Saved", type: "success" });
      queryClient.setQueryData(userKeys.detail(variables.id), user);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: userKeys.all }),
        queryClient.invalidateQueries({ queryKey: groupKeys.all }),
        queryClient.invalidateQueries({ queryKey: sessionQueryOptions.queryKey }),
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
        queryClient.invalidateQueries({ queryKey: userKeys.all }),
        queryClient.invalidateQueries({ queryKey: groupKeys.all }),
      ]);
    },
  });
}
