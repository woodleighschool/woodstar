import { useNavigate, useParams } from "@tanstack/react-router";

import { PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { Skeleton } from "@components/ui/skeleton";
import { LiveRunner } from "@features/osquery/live/live-runner";

import { useCheck } from "./queries";
export function CheckLivePage() {
  const navigate = useNavigate();
  const { id: checkId } = useParams({
    from: "/_authenticated/osquery/checks/$id",
  });
  const check = useCheck(Number(checkId));
  if (check.error) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load check"
          error={check.error}
          onRetry={() => void check.refetch()}
        />
      </PageShell>
    );
  }
  if (!check.data) {
    return (
      <PageShell>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full max-w-3xl" />
      </PageShell>
    );
  }
  return (
    <LiveRunner
      kind="check"
      itemId={Number(checkId)}
      sql={check.data.query}
      onCancel={() =>
        void navigate({
          to: "/osquery/checks/$id",
          params: { id: checkId },
        })
      }
    />
  );
}
