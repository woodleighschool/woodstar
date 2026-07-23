import { useNavigate, useParams } from "@tanstack/react-router";

import { LiveRunButton } from "@/components/queries/query-ui";
import { QueryGate } from "@/components/query-gate";
import { useReport, useUpdateReport } from "@/hooks/use-reports";
import { parseRouteID } from "@/lib/route-params";
import { ReportForm, reportFromDetail } from "@/pages/osquery/reports/fields";

export function ReportEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const reportId = params.id ?? "";
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
      title="Edit Report"
      submitLabel="Save"
      resultsReportId={id}
      headerActions={<LiveRunButton to="/osquery/reports/$id/live" params={{ id: reportId }} />}
      onSubmit={async (value) => (await update.mutateAsync(value)).id}
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({
            to: "/osquery/reports/$id",
            params: { id: String(savedID) },
          });
        }
      }}
    />
  );
}
