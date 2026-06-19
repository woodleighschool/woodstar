import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type {
  ApiError,
  PageExecutionEvent,
  PageFileAccessEvent,
  SantaExecutionEvent,
  SantaFileAccessEvent,
  SantaHostSummary,
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
import { nonEmpty } from "@/lib/utils";

export type { SantaFileAccessEvent, SantaHostSummary };
export type SantaEvent = SantaExecutionEvent;
export type SantaEventListResult = PageExecutionEvent;
export type SantaFileAccessEventListResult = PageFileAccessEvent;
export type SantaExecutionDecision = SantaEvent["decision"];
export type SantaFileAccessDecision = SantaFileAccessEvent["decision"];

export type SantaEventListParams = Omit<NonNullable<ListSantaEventsData["query"]>, "since">;
export type SantaFileAccessEventListParams = Omit<
  NonNullable<ListSantaFileAccessEventsData["query"]>,
  "since"
>;
export type SantaEventDecisionFilter = NonNullable<
  NonNullable<SantaEventListParams["decisions"]>[number]
>;

export function useSantaEvents(params: SantaEventListParams = {}) {
  const queryParams = {
    ...baseListParams(params),
    host_id: params.host_id,
    decisions: params.decisions && params.decisions.length > 0 ? params.decisions : undefined,
    user: nonEmpty(params.user),
  };

  return useQuery<SantaEventListResult, ApiError>({
    queryKey: queryKeys.santaEvents(queryParams),
    queryFn: ({ signal }) => unwrap(listSantaEvents({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaEvent(id: number | null) {
  return useQuery<SantaEvent, ApiError>({
    queryKey: queryKeys.santaEvent(id),
    queryFn: ({ signal }) =>
      unwrap(
        getSantaEvent({
          path: { id: id ?? 0 },
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

  return useQuery<SantaFileAccessEventListResult, ApiError>({
    queryKey: queryKeys.santaFileAccessEvents(queryParams),
    queryFn: ({ signal }) => unwrap(listSantaFileAccessEvents({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useSantaFileAccessEvent(id: number | null) {
  return useQuery<SantaFileAccessEvent, ApiError>({
    queryKey: queryKeys.santaFileAccessEvent(id),
    queryFn: ({ signal }) =>
      unwrap(
        getSantaFileAccessEvent({
          path: { id: id ?? 0 },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}
