import { useRouter } from "@tanstack/react-router";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { Button } from "@components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@components/ui/empty";

import { useOsqueryHistoryState } from "./history";
import { LiveRunner } from "./live-runner";

function OsqueryLivePage({ kind }: { kind: "report" | "check" }) {
  const router = useRouter();
  const historyState = useOsqueryHistoryState();
  const listPath = kind === "report" ? "/osquery/reports" : "/osquery/checks";
  const listLabel = kind === "report" ? "Reports" : "Checks";

  if (historyState?.view !== "live" || historyState.kind !== kind) {
    return (
      <PageShell>
        <PageHeader title="Run Live" />
        <Empty>
          <EmptyHeader>
            <EmptyTitle>Live query unavailable</EmptyTitle>
            <EmptyDescription>
              Its SQL is unavailable. Return to {listLabel.toLowerCase()} and run it again.
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button
              variant="outline"
              size="sm"
              render={<Link to={listPath} />}
              nativeButton={false}
            >
              View {listLabel}
            </Button>
          </EmptyContent>
        </Empty>
      </PageShell>
    );
  }

  return (
    <LiveRunner
      kind={historyState.kind}
      sql={historyState.sql}
      onCancel={() => {
        if (router.history.canGoBack()) {
          router.history.back();
          return;
        }
        void router.navigate({ to: listPath });
      }}
    />
  );
}

export function ReportLivePage() {
  return <OsqueryLivePage kind="report" />;
}

export function CheckLivePage() {
  return <OsqueryLivePage kind="check" />;
}
