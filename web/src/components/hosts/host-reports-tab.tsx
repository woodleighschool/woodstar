import { ClipboardList } from "lucide-react";
import { useMemo } from "react";

import { ReportResultCard } from "@/components/reports/report-result-card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { useHostReports } from "@/hooks/use-hosts";

export function HostReportsTab({ hostId, hostParam }: { hostId: number | null; hostParam: string }) {
  const reports = useHostReports(hostId);
  const reportItems = reports.data?.items;
  const rows = useMemo(() => reportItems ?? [], [reportItems]);

  if (reports.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to Load Reports</AlertTitle>
        <AlertDescription>{reports.error.message}</AlertDescription>
      </Alert>
    );
  }

  if (reports.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-48 rounded-lg" />
        <Skeleton className="h-48 rounded-lg" />
      </div>
    );
  }

  if (rows.length === 0) {
    return (
      <div className="rounded-lg border">
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <ClipboardList />
            </EmptyMedia>
            <EmptyTitle>No Reports</EmptyTitle>
            <EmptyDescription>Add a scheduled report to view custom vitals for this host.</EmptyDescription>
          </EmptyHeader>
        </Empty>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {rows.map((report) => (
        <ReportResultCard key={report.report_id} report={report} hostParam={hostParam} />
      ))}
    </div>
  );
}
