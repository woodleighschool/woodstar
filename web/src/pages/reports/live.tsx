import { Link, useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useReport } from "@/hooks/use-reports";
import { LiveRunner } from "@/pages/live-runner";

export function ReportLivePage() {
  const { reportId } = useParams({ from: "/_authenticated/reports/$reportId" });
  const report = useReport(reportId);

  if (report.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to load report</AlertTitle>
          <AlertDescription>{report.error.message}</AlertDescription>
        </Alert>
      </PageShell>
    );
  }
  if (!report.data) {
    return (
      <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading report...
      </PageShell>
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
