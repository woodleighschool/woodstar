import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useReducer } from "react";

import type {
  ApiError,
  OsqueryHandle,
  OsqueryLiveQueryCreateBody,
  OsqueryLiveQuerySnapshotEvent,
  OsqueryLiveQueryTargetCountBody,
  OsqueryLiveQueryTargetCountOutputBody,
} from "@lib/api";
import {
  countLiveQueryTargets,
  createLiveQuery,
  deleteLiveQuery,
  streamLiveQuery,
  unwrap,
} from "@lib/api";
const LIVE_QUERY_TARGET_REFRESH_MS = 30_000;

export type LiveQueryHandle = OsqueryHandle;
export type LiveQuerySnapshot = OsqueryLiveQuerySnapshotEvent;
export type {
  OsqueryLiveQueryCreateBody,
  OsqueryLiveQueryTargetCountBody,
  OsqueryLiveQueryTargetCountOutputBody,
};

interface StreamState {
  snapshots: LiveQuerySnapshot[];
  status: "idle" | "running" | "completed" | "error";
  error?: string;
}

type StreamAction =
  | { type: "running" }
  | { type: "snapshot"; snapshot: LiveQuerySnapshot }
  | { type: "completed" }
  | { type: "error"; message: string }
  | { type: "reset" };

function streamReducer(state: StreamState, action: StreamAction): StreamState {
  switch (action.type) {
    case "running":
      return { ...state, status: "running", error: undefined };
    case "snapshot":
      return {
        ...state,
        snapshots: upsertSnapshot(state.snapshots, action.snapshot),
      };
    case "completed":
      return { ...state, status: "completed" };
    case "error":
      return { ...state, status: "error", error: action.message };
    case "reset":
      return { snapshots: [], status: "idle" };
  }
  return assertNever(action);
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
    queryKey: ["osquery", "live-queries", "target-count", body],
    queryFn: ({ signal }) => unwrap(countLiveQueryTargets({ body, signal })),
    enabled,
    refetchInterval: LIVE_QUERY_TARGET_REFRESH_MS,
  });
}

// useLiveQueryStream keeps the latest streamed snapshot for each targeted host
// until the server publishes `completed`.
export function useLiveQueryStream(liveQueryId: number | null) {
  const [state, dispatch] = useReducer(streamReducer, {
    snapshots: [],
    status: "idle",
  });

  useEffect(() => {
    if (liveQueryId === null) return undefined;
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
          case "snapshot":
            dispatch({ type: "snapshot", snapshot: event });
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

function upsertSnapshot(
  snapshots: LiveQuerySnapshot[],
  next: LiveQuerySnapshot,
): LiveQuerySnapshot[] {
  const index = snapshots.findIndex((snapshot) => snapshot.host_id === next.host_id);
  const updated =
    index === -1
      ? [...snapshots, next]
      : snapshots.map((snapshot, snapshotIndex) => (snapshotIndex === index ? next : snapshot));
  return updated.toSorted(
    (a, b) =>
      Date.parse(b.updated_at) - Date.parse(a.updated_at) ||
      a.host_name.localeCompare(b.host_name) ||
      a.host_id - b.host_id,
  );
}

function assertNever(value: never): never {
  throw new Error(`Unexpected live query event: ${JSON.stringify(value)}`);
}
