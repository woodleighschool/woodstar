import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useReducer } from "react";

import type {
  ApiError,
  OsqueryHandle,
  OsqueryLiveQueryCreateBody,
  OsqueryLiveQueryResultEvent,
  OsqueryLiveQueryTargetCountBody,
  OsqueryLiveQueryTargetCountOutputBody,
} from "@/lib/api";
import { countLiveQueryTargets, createLiveQuery, stopLiveQuery, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

const LIVE_QUERY_TARGET_REFRESH_MS = 30_000;

export type LiveQueryHandle = OsqueryHandle;
export type LiveQueryResult = OsqueryLiveQueryResultEvent;
export type {
  OsqueryLiveQueryCreateBody,
  OsqueryLiveQueryTargetCountBody,
  OsqueryLiveQueryTargetCountOutputBody,
};

export type LiveQueryRow = LiveQueryResult & {
  // monotonic per-stream id for stable React keys; not from the server
  _seq: number;
};

interface StreamState {
  results: LiveQueryRow[];
  nextSeq: number;
  status: "idle" | "running" | "completed" | "error";
  error?: string;
}

type StreamAction =
  | { type: "running" }
  | { type: "result"; result: LiveQueryResult }
  | { type: "completed" }
  | { type: "error"; message: string }
  | { type: "reset" };

function streamReducer(state: StreamState, action: StreamAction): StreamState {
  switch (action.type) {
    case "running":
      return { ...state, status: "running", error: undefined };
    case "result":
      return {
        ...state,
        results: [...state.results, { ...action.result, _seq: state.nextSeq }],
        nextSeq: state.nextSeq + 1,
      };
    case "completed":
      return { ...state, status: "completed" };
    case "error":
      return { ...state, status: "error", error: action.message };
    case "reset":
      return { results: [], nextSeq: 0, status: "idle" };
  }
}

export function useCreateLiveQuery() {
  return useMutation<LiveQueryHandle, ApiError, OsqueryLiveQueryCreateBody>({
    mutationFn: (body) => unwrap(createLiveQuery({ body })),
  });
}

export function useStopLiveQuery() {
  return useMutation<void, ApiError, number>({
    mutationFn: async (id) => {
      await unwrap(stopLiveQuery({ path: { id } }));
    },
  });
}

export function useLiveQueryTargetCount(body: OsqueryLiveQueryTargetCountBody, enabled: boolean) {
  return useQuery<OsqueryLiveQueryTargetCountOutputBody, ApiError>({
    queryKey: queryKeys.liveQueryTargetCount(body),
    queryFn: ({ signal }) => unwrap(countLiveQueryTargets({ body, signal })),
    enabled,
    refetchInterval: LIVE_QUERY_TARGET_REFRESH_MS,
  });
}

// useLiveQueryStream opens an EventSource against /api/live-queries/{id}/stream
// and accumulates result events until the server publishes `completed`.
export function useLiveQueryStream(liveQueryId: number | null) {
  const [state, dispatch] = useReducer(streamReducer, { results: [], nextSeq: 0, status: "idle" });

  useEffect(() => {
    if (liveQueryId === null) return;
    dispatch({ type: "reset" });
    dispatch({ type: "running" });

    const source = new EventSource(`/api/live-queries/${encodeURIComponent(liveQueryId)}/stream`);
    let completed = false;
    source.addEventListener("open", () => dispatch({ type: "running" }));
    source.addEventListener("result", (event: MessageEvent<string>) => {
      try {
        const parsed = JSON.parse(event.data) as LiveQueryResult;
        dispatch({ type: "result", result: parsed });
      } catch {
        // server controls the schema; drop malformed payloads silently
      }
    });
    source.addEventListener("completed", () => {
      completed = true;
      dispatch({ type: "completed" });
      source.close();
    });
    source.addEventListener("ping", () => {
      // heartbeat
    });
    source.onerror = () => {
      if (completed) return;
      source.close();
      dispatch({ type: "error", message: "stream interrupted" });
    };

    return () => source.close();
  }, [liveQueryId]);

  return state;
}
