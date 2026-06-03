import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { ApiError, EntraDepartment, EntraGroup, EntraUser, Page } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { EntraDepartment, EntraGroup, EntraUser };
export type EntraUsersResult = Page<EntraUser>;
export type EntraGroupsResult = Page<EntraGroup>;
export type EntraDepartmentsResult = Page<EntraDepartment>;

export interface EntraListParams {
  q?: string;
  page_index?: number;
  page_size?: number;
  sort?: string;
  values?: string[];
}

function entraQueryParams(params: EntraListParams = {}) {
  return {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
    values: params.values && params.values.length > 0 ? params.values : undefined,
  };
}

export function useEntraUsers(params: EntraListParams = {}) {
  const queryParams = entraQueryParams(params);
  return useQuery<EntraUsersResult, ApiError>({
    queryKey: queryKeys.entraUsers(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/entra/users", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useEntraGroups(params: EntraListParams = {}) {
  const queryParams = entraQueryParams(params);
  return useQuery<EntraGroupsResult, ApiError>({
    queryKey: queryKeys.entraGroups(queryParams),
    queryFn: ({ signal }) => unwrap(apiClient.GET("/api/entra/groups", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useEntraDepartments(params: EntraListParams = {}) {
  const queryParams = entraQueryParams(params);
  return useQuery<EntraDepartmentsResult, ApiError>({
    queryKey: queryKeys.entraDepartments(queryParams),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/entra/departments", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}
