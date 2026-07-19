import { queryOptions } from "@tanstack/react-query";

import type {
  ApiError,
  SantaConfiguration,
  SantaExecutionEvent,
  SantaFileAccessEvent,
  SantaRule,
} from "@/lib/api";
import {
  getSantaConfiguration,
  getSantaEvent,
  getSantaFileAccessEvent,
  getSantaRule,
  unwrap,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export function santaConfigurationQueryOptions(id: number | null) {
  return queryOptions<SantaConfiguration, ApiError>({
    queryKey: queryKeys.santaConfiguration(id),
    queryFn: ({ signal }) => unwrap(getSantaConfiguration({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function santaRuleQueryOptions(id: number | null) {
  return queryOptions<SantaRule, ApiError>({
    queryKey: queryKeys.santaRule(id),
    queryFn: ({ signal }) => unwrap(getSantaRule({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function santaEventQueryOptions(id: number | null) {
  return queryOptions<SantaExecutionEvent, ApiError>({
    queryKey: queryKeys.santaEvent(id),
    queryFn: ({ signal }) => unwrap(getSantaEvent({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function santaFileAccessEventQueryOptions(id: number | null) {
  return queryOptions<SantaFileAccessEvent, ApiError>({
    queryKey: queryKeys.santaFileAccessEvent(id),
    queryFn: ({ signal }) => unwrap(getSantaFileAccessEvent({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}
