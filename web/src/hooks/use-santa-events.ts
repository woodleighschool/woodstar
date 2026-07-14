import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type {
  ApiError,
  PageExecutionEvent,
  PageFileAccessEvent,
  SantaExecutionEvent,
  SantaFileAccessEvent,
} from "@/lib/api";
import {
  getSantaEvent,
  getSantaFileAccessEvent,
  listSantaEvents,
  listSantaFileAccessEvents,
  unwrap,
} from "@/lib/api";
import type {
  ListSantaEventsData,
  ListSantaFileAccessEventsData,
} from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";
import { nonEmpty } from "@/lib/utils";

export type SantaEventListParams = Omit<NonNullable<ListSantaEventsData["query"]>, "since">;
export type SantaFileAccessEventListParams = Omit<
  NonNullable<ListSantaFileAccessEventsData["query"]>,
  "since"
>;
export type SantaEventDecisionFilter = NonNullable<
  NonNullable<SantaEventListParams["decisions"]>[number]
>;

const SANTA_EVENT_REFRESH_MS = 30_000;

export function useSantaEvents(params: SantaEventListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    host_id: params.host_id,
    decisions: params.decisions && params.decisions.length > 0 ? params.decisions : undefined,
    user: nonEmpty(params.user),
  };

  return useQuery<PageExecutionEvent, ApiError>({
    queryKey: queryKeys.santaEvents(queryParams),
    queryFn: ({ signal }) => unwrap(listSantaEvents({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: SANTA_EVENT_REFRESH_MS,
  });
}

export function useSantaEvent(id: number | null) {
  return useQuery<SantaExecutionEvent, ApiError>({
    queryKey: queryKeys.santaEvent(id),
    queryFn: ({ signal }) =>
      unwrap(
        getSantaEvent({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useSantaFileAccessEvents(params: SantaFileAccessEventListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    host_id: params.host_id,
    decisions: params.decisions && params.decisions.length > 0 ? params.decisions : undefined,
  };

  return useQuery<PageFileAccessEvent, ApiError>({
    queryKey: queryKeys.santaFileAccessEvents(queryParams),
    queryFn: ({ signal }) => unwrap(listSantaFileAccessEvents({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
    refetchInterval: SANTA_EVENT_REFRESH_MS,
  });
}

export function useSantaFileAccessEvent(id: number | null) {
  return useQuery<SantaFileAccessEvent, ApiError>({
    queryKey: queryKeys.santaFileAccessEvent(id),
    queryFn: ({ signal }) =>
      unwrap(
        getSantaFileAccessEvent({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
}
