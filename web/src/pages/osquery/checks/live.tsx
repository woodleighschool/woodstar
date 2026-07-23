import { Link, useParams } from "@tanstack/react-router";

import { PageShell } from "@/components/layout/page-layout";
import { LiveRunner } from "@/components/osquery/live-runner";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCheck } from "@/hooks/use-checks";
export function CheckLivePage() {
  const { checkId } = useParams({ from: "/_authenticated/osquery/checks/$checkId" });
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
      editAction={
        <Button
          variant="outline"
          size="sm"
          render={<Link to="/osquery/checks/$checkId" params={{ checkId }} />}
          nativeButton={false}
        >
          Edit Check
        </Button>
      }
    />
  );
}
