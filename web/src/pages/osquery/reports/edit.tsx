import { useNavigate, useParams } from "@tanstack/react-router";

import { LiveRunButton } from "@/components/queries/query-ui";
import { QueryGate } from "@/components/query-gate";
import { useReport, useUpdateReport } from "@/hooks/use-reports";
import { parseRouteID } from "@/lib/route-params";
import { ReportForm, reportFromDetail } from "@/pages/osquery/reports/fields";

export function ReportEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const reportId = params.reportId ?? "";
  const id = parseRouteID(reportId);
  const detail = useReport(id);
  const update = useUpdateReport(id);

  if (id === null) {
    return (
      <QueryGate title="Failed to load report" error={{ message: "Report route is invalid." }} />
    );
  }

  if (detail.error || !detail.data) {
    return (
      <QueryGate
        title="Failed to load report"
        error={detail.error}
        onRetry={() => void detail.refetch()}
      />
    );
  }

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
