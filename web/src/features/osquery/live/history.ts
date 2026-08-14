import { type HistoryState, useRouter, useRouterState } from "@tanstack/react-router";
import { useCallback } from "react";

import type { OsqueryPolicyMutation, OsqueryReportMutation } from "@lib/api";

type ReportFormHistoryState = {
  view: "report-form";
  id?: number;
  value: OsqueryReportMutation;
};

type PolicyFormHistoryState = {
  view: "policy-form";
  id?: number;
  value: OsqueryPolicyMutation;
};

type OsqueryFormHistoryState = ReportFormHistoryState | PolicyFormHistoryState;

type OsqueryLiveHistoryState = {
  view: "live";
  kind: "report" | "policy";
  sql: string;
};

type OpenOsqueryLiveOptions =
  | {
      kind: "report";
      sql: string;
      id?: number;
      form?: ReportFormHistoryState;
    }
  | {
      kind: "policy";
      sql: string;
      id?: number;
      form?: PolicyFormHistoryState;
    };

type OsqueryHistoryState = OsqueryFormHistoryState | OsqueryLiveHistoryState;

declare module "@tanstack/react-router" {
  interface HistoryState {
    osquery?: OsqueryHistoryState;
  }
}

export function useOsqueryHistoryState() {
  return useRouterState({ select: (state) => state.location.state.osquery });
}

export function useOpenOsqueryLive() {
  const router = useRouter();

  return useCallback(
    async ({ kind, sql, id, form }: OpenOsqueryLiveOptions) => {
      if (form) {
        const current = router.state.location;
        await router.navigate({
          href: current.href,
          replace: true,
          state: (previous) => ({ ...previous, osquery: form }),
        });
      }
      const state = (previous: HistoryState) => ({
        ...previous,
        osquery: { view: "live", kind, sql } satisfies OsqueryLiveHistoryState,
      });
      if (kind === "report") {
        if (id === undefined) {
          await router.navigate({ to: "/osquery/reports/new/live", state });
          return;
        }
        await router.navigate({
          to: "/osquery/reports/$id/live",
          params: { id: String(id) },
          state,
        });
        return;
      }
      if (id === undefined) {
        await router.navigate({ to: "/osquery/policies/new/live", state });
        return;
      }
      await router.navigate({
        to: "/osquery/policies/$id/live",
        params: { id: String(id) },
        state,
      });
    },
    [router],
  );
}

export function useClearOsqueryHistoryState() {
  const router = useRouter();

  return useCallback(async () => {
    if (router.state.location.state.osquery === undefined) return;
    const current = router.state.location;
    await router.navigate({
      href: current.href,
      replace: true,
      state: (previous) => ({ ...previous, osquery: undefined }),
    });
  }, [router]);
}
