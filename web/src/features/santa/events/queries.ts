import { keepPreviousData, queryOptions, useQuery } from "@tanstack/react-query";

import type {
  ApiError,
  PageExecutionEvent,
  PageFileAccessEvent,
  SantaExecutionEvent,
  SantaFileAccessEvent,
} from "@lib/api";
import {
  getSantaEvent,
  getSantaFileAccessEvent,
  listSantaEvents,
  listSantaFileAccessEvents,
  unwrap,
} from "@lib/api";
import type { ListSantaEventsData, ListSantaFileAccessEventsData } from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { nonEmpty } from "@lib/utils";

type QueryParams = Record<string, unknown>;

type SantaEventListParams = Omit<NonNullable<ListSantaEventsData["query"]>, "since">;
type SantaFileAccessEventListParams = Omit<
  NonNullable<ListSantaFileAccessEventsData["query"]>,
  "since"
>;
export type SantaEventDecisionFilter = NonNullable<
  NonNullable<SantaEventListParams["decisions"]>[number]
>;

const SANTA_EVENT_REFRESH_MS = 30_000;

const eventKeys = {
  executions: (params?: QueryParams) =>
    ["santa", "events", "executions", "list", params ?? {}] as const,
  execution: (id: number) => ["santa", "events", "executions", "detail", id] as const,
  fileAccess: (params?: QueryParams) =>
    ["santa", "events", "file-access", "list", params ?? {}] as const,
  fileAccessDetail: (id: number) => ["santa", "events", "file-access", "detail", id] as const,
};

export function santaEventQueryOptions(id: number) {
  return queryOptions<SantaExecutionEvent, ApiError>({
    queryKey: eventKeys.execution(id),
    queryFn: ({ signal }) => unwrap(getSantaEvent({ path: { id }, signal })),
  });
}

export function santaFileAccessEventQueryOptions(id: number) {
  return queryOptions<SantaFileAccessEvent, ApiError>({
    queryKey: eventKeys.fileAccessDetail(id),
    queryFn: ({ signal }) => unwrap(getSantaFileAccessEvent({ path: { id }, signal })),
  });
}

export function useSantaEvents(params: SantaEventListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    host_id: params.host_id,
    decisions: params.decisions && params.decisions.length > 0 ? params.decisions : undefined,
    user: nonEmpty(params.user),
  };

  return useQuery<PageExecutionEvent, ApiError>({
    queryKey: eventKeys.executions(queryParams),
    queryFn: ({ signal }) => unwrap(listSantaEvents({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: SANTA_EVENT_REFRESH_MS,
  });
}

export function useSantaEvent(id: number) {
  return useQuery(santaEventQueryOptions(id));
}

export function useSantaFileAccessEvents(params: SantaFileAccessEventListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    host_id: params.host_id,
    decisions: params.decisions && params.decisions.length > 0 ? params.decisions : undefined,
  };

  return useQuery<PageFileAccessEvent, ApiError>({
    queryKey: eventKeys.fileAccess(queryParams),
    queryFn: ({ signal }) => unwrap(listSantaFileAccessEvents({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: SANTA_EVENT_REFRESH_MS,
  });
}

export function useSantaFileAccessEvent(id: number) {
  return useQuery(santaFileAccessEventQueryOptions(id));
}
