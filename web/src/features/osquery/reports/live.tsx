import { useNavigate, useParams } from "@tanstack/react-router";

import { PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { Skeleton } from "@components/ui/skeleton";
import { LiveRunner } from "@features/osquery/live/live-runner";

import { useReport } from "./queries";
export function ReportLivePage() {
  const navigate = useNavigate();
  const { id: reportId } = useParams({
    from: "/_authenticated/osquery/reports/$id",
  });
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
      onCancel={() =>
        void navigate({
          to: "/osquery/reports/$id",
          params: { id: reportId },
        })
      }
    />
  );
}
