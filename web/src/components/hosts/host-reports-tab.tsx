import { useMemo } from "react";

import { EmptyPanel } from "@/components/empty-panel";
import { QueryError } from "@/components/query-error";
import { ReportResultCard } from "@/components/reports/report-result-card";
import { useHostReports } from "@/hooks/use-hosts";

export function HostReportsTab({ hostId }: { hostId: number | null }) {
  const reports = useHostReports(hostId);
  const reportItems = reports.data;
  const rows = useMemo(() => reportItems ?? [], [reportItems]);

  if (reports.error) {
    return <QueryError title="Failed to load reports" error={reports.error} onRetry={() => void reports.refetch()} />;
  }

  if (reports.isLoading) {
    return null;
  }

  if (rows.length === 0) {
    return <EmptyPanel>No reports yet</EmptyPanel>;
  }

  return (
    <div className="flex flex-col gap-4">
      {rows.map((report) => (
        <ReportResultCard key={report.report_id} report={report} />
      ))}
    </div>
  );
}
