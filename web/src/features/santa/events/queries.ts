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
import { detailPath } from "@lib/route-params";
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
  execution: (id: number | null) => ["santa", "events", "executions", "detail", id] as const,
  fileAccess: (params?: QueryParams) =>
    ["santa", "events", "file-access", "list", params ?? {}] as const,
  fileAccessDetail: (id: number | null) =>
    ["santa", "events", "file-access", "detail", id] as const,
};

export function santaEventQueryOptions(id: number | null) {
  return queryOptions<SantaExecutionEvent, ApiError>({
    queryKey: eventKeys.execution(id),
    queryFn: ({ signal }) => unwrap(getSantaEvent({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function santaFileAccessEventQueryOptions(id: number | null) {
  return queryOptions<SantaFileAccessEvent, ApiError>({
    queryKey: eventKeys.fileAccessDetail(id),
    queryFn: ({ signal }) => unwrap(getSantaFileAccessEvent({ path: detailPath(id), signal })),
    enabled: id !== null,
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

export function useSantaEvent(id: number | null) {
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

export function useSantaFileAccessEvent(id: number | null) {
  return useQuery(santaFileAccessEventQueryOptions(id));
}
