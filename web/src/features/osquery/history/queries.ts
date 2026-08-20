import { useQuery } from "@tanstack/react-query";

import type { ApiError, OsqueryHostStatusPoint, OsqueryPolicyStatusPoint } from "@lib/api";
import { listOsqueryHostStatusHistory, listOsqueryPolicyStatusHistory, unwrap } from "@lib/api";
import { detailPath } from "@lib/route-params";

export type HistoryRange = "24h" | "7d" | "30d";

export const HISTORY_REFRESH_MS = 5 * 60_000;
const RANGE_MS: Record<HistoryRange, number> = {
  "24h": 24 * 60 * 60_000,
  "7d": 7 * 24 * 60 * 60_000,
  "30d": 30 * 24 * 60 * 60_000,
};

export function historyBounds(range: HistoryRange, now = Date.now()): [number, number] {
  const bucketedNow = Math.floor(now / HISTORY_REFRESH_MS) * HISTORY_REFRESH_MS;
  return [bucketedNow - RANGE_MS[range], bucketedNow];
}

function since(range: HistoryRange): string {
  return new Date(historyBounds(range)[0]).toISOString();
}

export function useHostStatusHistory(range: HistoryRange) {
  return useQuery<OsqueryHostStatusPoint[], ApiError>({
    queryKey: ["osquery", "history", "hosts", range],
    queryFn: ({ signal }) =>
      unwrap(listOsqueryHostStatusHistory({ query: { since: since(range) }, signal })),
    refetchInterval: HISTORY_REFRESH_MS,
  });
}

export function usePolicyStatusHistory(id: number | null, range: HistoryRange) {
  return useQuery<OsqueryPolicyStatusPoint[], ApiError>({
    queryKey: ["osquery", "policies", "detail", id, "history", range],
    queryFn: ({ signal }) =>
      unwrap(
        listOsqueryPolicyStatusHistory({
          path: detailPath(id),
          query: { since: since(range) },
          signal,
        }),
      ),
    enabled: id !== null,
    refetchInterval: HISTORY_REFRESH_MS,
  });
}
