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
import {
  countLiveQueryTargets,
  createLiveQuery,
  deleteLiveQuery,
  streamLiveQuery,
  unwrap,
} from "@/lib/api";
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
      await unwrap(deleteLiveQuery({ path: { id } }));
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

// useLiveQueryStream accumulates generated stream events until the server
// publishes `completed`.
export function useLiveQueryStream(liveQueryId: number | null) {
  const [state, dispatch] = useReducer(streamReducer, { results: [], nextSeq: 0, status: "idle" });

  useEffect(() => {
    if (liveQueryId === null) return;
    dispatch({ type: "reset" });
    dispatch({ type: "running" });

    const abortController = new AbortController();
    let terminal = false;

    const fail = () => {
      if (terminal || abortController.signal.aborted) return;
      terminal = true;
      dispatch({ type: "error", message: "stream interrupted" });
    };

    const consume = async () => {
      const { stream } = await streamLiveQuery({
        path: { id: liveQueryId },
        headers: { Accept: "text/event-stream" },
        signal: abortController.signal,
        sseMaxRetryAttempts: 1,
        onSseError: fail,
      });

      for await (const event of stream) {
        if (abortController.signal.aborted) return;
        switch (event.type) {
          case "ping":
            break;
          case "result":
            dispatch({ type: "result", result: event });
            break;
          case "completed":
            terminal = true;
            dispatch({ type: "completed" });
            abortController.abort();
            return;
          default:
            assertNever(event);
        }
      }

      fail();
    };

    void consume().catch(fail);
    return () => abortController.abort();
  }, [liveQueryId]);

  return state;
}

function assertNever(value: never): never {
  throw new Error(`Unexpected live query event: ${JSON.stringify(value)}`);
}
