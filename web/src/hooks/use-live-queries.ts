import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useReducer } from "react";

import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";

export type LiveQueryHandle = Schemas["LiveQueryHandleBody"];
export type LiveQueryCreate = Schemas["LiveQueryCreateBody"];
export type LiveQueryResult = Schemas["LiveQueryResultEvent"];
export type LiveQueryTargetCount = Schemas["LiveQueryTargetCountOutputBody"];
export type LiveQueryTargetCountBody = Schemas["LiveQueryTargetCountBody"];

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
  return useMutation<LiveQueryHandle, ApiError, LiveQueryCreate>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/live-queries", { body })),
  });
}

export function useStopLiveQuery() {
  return useMutation<void, ApiError, number>({
    mutationFn: async (id) => {
      await unwrap(apiClient.POST("/api/live-queries/{id}/stop", { params: { path: { id } } }));
    },
  });
}

export function useLiveQueryTargetCount(body: LiveQueryTargetCountBody, enabled: boolean) {
  const hosts = body.selected?.hosts ?? [];
  const labels = body.selected?.labels ?? [];
  return useQuery<LiveQueryTargetCount, ApiError>({
    queryKey: ["live-query-target-count", body.report_id ?? null, hosts, labels],
    queryFn: () => unwrap(apiClient.POST("/api/live-queries/targets/count", { body })),
    enabled,
  });
}

// useLiveQueryStream opens an EventSource against /api/live-queries/{id}/stream
// and accumulates result events until the server publishes `completed`.
export function useLiveQueryStream(liveQueryId: string) {
  const [state, dispatch] = useReducer(streamReducer, { results: [], nextSeq: 0, status: "idle" });

  useEffect(() => {
    if (liveQueryId === "") return;
    dispatch({ type: "reset" });
    dispatch({ type: "running" });

    const source = new EventSource(`/api/live-queries/${encodeURIComponent(liveQueryId)}/stream`);
    source.addEventListener("open", () => dispatch({ type: "running" }));
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
