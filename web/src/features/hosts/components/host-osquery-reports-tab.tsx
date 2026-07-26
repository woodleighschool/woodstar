import { useMemo } from "react";

import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { useHostOsqueryReports } from "@features/hosts/queries";
import { ReportResultCard } from "@features/osquery/reports/report-result-card";

export function HostOsqueryReportsTab({ hostId }: { hostId: number | null }) {
  const reports = useHostOsqueryReports(hostId);
  const reportItems = reports.data;
  const rows = useMemo(() => reportItems ?? [], [reportItems]);

  if (reports.error) {
    return (
      <QueryError
        title="Failed to load reports"
        error={reports.error}
        onRetry={() => void reports.refetch()}
      />
    );
  }

  if (reports.isLoading) {
    return null;
  }

  if (rows.length === 0) {
    return <PanelEmptyState>No reports yet</PanelEmptyState>;
  }

  return (
    <div className="flex flex-col gap-4">
      {rows.map((report) => (
        <ReportResultCard key={report.report_id} report={report} />
      ))}
    </div>
  );
}
