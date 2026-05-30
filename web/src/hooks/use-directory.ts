import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { ApiError, Department, DirectoryGroup, DirectoryUser, Page } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { DirectoryGroup, DirectoryUser };
export type DirectoryDepartment = Department;
export type DirectoryUsersResult = Page<DirectoryUser>;
export type DirectoryGroupsResult = Page<DirectoryGroup>;
export type DirectoryDepartmentsResult = Page<DirectoryDepartment>;

export interface DirectoryListParams {
  q?: string;
  page_index?: number;
  page_size?: number;
  sort?: string;
  values?: string[];
}

function directoryQueryParams(params: DirectoryListParams = {}) {
  return {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
    values: params.values && params.values.length > 0 ? params.values : undefined,
  };
}

export function useDirectoryUsers(params: DirectoryListParams = {}) {
  const queryParams = directoryQueryParams(params);
  return useQuery<DirectoryUsersResult, ApiError>({
    queryKey: queryKeys.directoryUsers(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/directory/users", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useDirectoryGroups(params: DirectoryListParams = {}) {
  const queryParams = directoryQueryParams(params);
  return useQuery<DirectoryGroupsResult, ApiError>({
    queryKey: queryKeys.directoryGroups(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/directory/groups", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useDirectoryDepartments(params: DirectoryListParams = {}) {
  const queryParams = directoryQueryParams(params);
  return useQuery<DirectoryDepartmentsResult, ApiError>({
    queryKey: queryKeys.directoryDepartments(queryParams),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/directory/departments", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}
