import { useMutation } from "@tanstack/react-query";
import { useEffect, useReducer } from "react";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";

export type LiveQueryHandle = Schemas["LiveQueryHandleBody"];
export type LiveQueryCreate = Schemas["LiveQueryCreateBody"];
export type LiveQueryResult = Schemas["LiveQueryResultEvent"];

export type LiveQueryRow = LiveQueryResult & {
  // monotonic per-stream id for stable React keys; not from the server
  _seq: number;
};

interface StreamState {
  results: LiveQueryRow[];
  nextSeq: number;
  status: "idle" | "open" | "completed" | "error";
  error?: string;
}

type StreamAction =
  | { type: "open" }
  | { type: "result"; result: LiveQueryResult }
  | { type: "completed" }
  | { type: "error"; message: string }
  | { type: "reset" };

function streamReducer(state: StreamState, action: StreamAction): StreamState {
  switch (action.type) {
    case "open":
      return { ...state, status: "open", error: undefined };
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
  return useMutation<LiveQueryHandle, ApiError, LiveQueryCreate>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/live-queries", { body })),
  });
}

// useLiveQueryStream opens an EventSource against /api/live-queries/{id}/stream
// and accumulates result events until the server publishes `completed`.
export function useLiveQueryStream(liveQueryId: string) {
  const [state, dispatch] = useReducer(streamReducer, { results: [], nextSeq: 0, status: "idle" });

  useEffect(() => {
    if (liveQueryId === "") return;
    dispatch({ type: "reset" });

    const source = new EventSource(`/api/live-queries/${encodeURIComponent(liveQueryId)}/stream`);
    source.addEventListener("open", () => dispatch({ type: "open" }));
    source.addEventListener("result", (event: MessageEvent<string>) => {
      try {
        const parsed = JSON.parse(event.data) as LiveQueryResult;
        dispatch({ type: "result", result: parsed });
      } catch {
        // server controls the schema — drop malformed payloads silently
      }
    });
    source.addEventListener("completed", () => {
      dispatch({ type: "completed" });
      source.close();
    });
    source.addEventListener("ping", () => {
      // heartbeat
    });
    source.onerror = () => {
      dispatch({ type: "error", message: "stream interrupted" });
    };

    return () => source.close();
  }, [liveQueryId]);

  return state;
}
