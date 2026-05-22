import { Link, useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useReport } from "@/hooks/use-reports";
import { LiveRunner } from "@/pages/live-runner";

export function ReportLivePage() {
  const { reportId } = useParams({ from: "/_authenticated/reports/$reportId" });
  const report = useReport(reportId);

  if (report.error) {
    return (
      <div className="p-6">
        <Alert variant="destructive">
          <AlertTitle>Failed to load report</AlertTitle>
          <AlertDescription>{report.error.message}</AlertDescription>
        </Alert>
      </div>
    );
  }
  if (!report.data) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading report...
      </div>
    );
  }

  return (
    <LiveRunner
      kind="report"
      itemId={reportId}
      name={report.data.name}
      sql={report.data.query}
      editAction={
        <Button asChild variant="outline" size="sm">
          <Link to="/reports/$reportId/edit" params={{ reportId }}>
            Edit report
          </Link>
        </Button>
      }
    />
  );
}
