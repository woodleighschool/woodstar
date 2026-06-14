import { useNavigate, useParams } from "@tanstack/react-router";

import { PageShell } from "@/components/layout/page-layout";
import { LiveRunButton } from "@/components/queries/query-ui";
import { QueryError } from "@/components/query-error";
import { useReport, useUpdateReport } from "@/hooks/use-reports";
import { ReportForm, reportFromDetail } from "@/pages/osquery/reports/fields";

export function ReportEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const reportId = params.reportId ?? "";
  const id = Number(reportId);
  const detail = useReport(id);
  const update = useUpdateReport(id);

  if (detail.error) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load report"
          error={detail.error}
          onRetry={() => void detail.refetch()}
        />
      </PageShell>
    );
  }
  if (!detail.data) return null;

  const report = detail.data;
  return (
    <ReportForm
      key={report.id}
      initial={reportFromDetail(report)}
      submitLabel="Save"
      resultsReportId={id}
      headerActions={<LiveRunButton to="/osquery/reports/$reportId/live" params={{ reportId }} />}
      onSubmit={async (value) => {
        const saved = await update.mutateAsync(value);
        void navigate({ to: "/osquery/reports/$reportId", params: { reportId: String(saved.id) } });
      }}
    />
  );
}
