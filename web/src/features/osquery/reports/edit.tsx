import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { parseRouteID } from "@lib/route-params";

import { ReportForm, reportFromDetail } from "./fields";
import { useReport, useUpdateReport } from "./queries";

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
      confirmQueryChange
      onCancel={() =>
        void navigate({
          to: "/osquery/reports/$id",
          params: { id: String(report.id) },
        })
      }
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
