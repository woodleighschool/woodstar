import { Link, useParams } from "@tanstack/react-router";

import { PageShell } from "@/components/layout/page-layout";
import { LiveRunner } from "@/components/osquery/live-runner";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useReport } from "@/hooks/use-reports";
export function ReportLivePage() {
  const { reportId } = useParams({ from: "/_authenticated/osquery/reports/$reportId" });
  const report = useReport(Number(reportId));
  if (report.error) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load report"
          error={report.error}
          onRetry={() => void report.refetch()}
        />
      </PageShell>
    );
  }
  if (!report.data) {
    return (
      <PageShell>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full max-w-3xl" />
      </PageShell>
    );
  }
  return (
    <LiveRunner
      kind="report"
      itemId={Number(reportId)}
      sql={report.data.query}
      editAction={
        <Button
          variant="outline"
          size="sm"
          render={<Link to="/osquery/reports/$reportId" params={{ reportId }} />}
          nativeButton={false}
        >
          Edit Report
        </Button>
      }
    />
  );
}
