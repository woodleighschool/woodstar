import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type {
  ApiError,
  Page,
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
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { SantaFileAccessEvent, SantaHostSummary };
export type SantaEvent = SantaExecutionEvent;
export type SantaEventListResult = Page<SantaEvent>;
export type SantaFileAccessEventListResult = Page<SantaFileAccessEvent>;
export type SantaExecutionDecision = SantaEvent["decision"];
export type SantaFileAccessDecision = SantaFileAccessEvent["decision"];

export type SantaEventListParams = NonNullable<ListSantaEventsData["query"]>;
export type SantaFileAccessEventListParams = NonNullable<ListSantaFileAccessEventsData["query"]>;
export type SantaEventDecisionFilter = NonNullable<
  NonNullable<SantaEventListParams["decisions"]>[number]
>;

export function useSantaEvents(params: SantaEventListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    host_id: params.host_id,
    decisions: params.decisions && params.decisions.length > 0 ? params.decisions : undefined,
    since: nonEmpty(params.since),
    user: nonEmpty(params.user),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
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
    q: nonEmpty(params.q),
    host_id: params.host_id,
    decisions: params.decisions && params.decisions.length > 0 ? params.decisions : undefined,
    since: nonEmpty(params.since),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
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
