import { useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2 } from "lucide-react";

import { DataTable, DataTableColumnHeader } from "@/components/data-table";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditButton, LiveRunButton, ShowQueryButton } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { useReport, useReportResults } from "@/hooks/use-reports";
import {
  reportRows,
  reportTableColumns,
  resultColumnNames,
  resultValue,
  type ReportTableRow,
} from "@/lib/query-results";

export function ReportDetailPage() {
  const { reportId } = useParams({ from: "/_authenticated/osquery/reports/$reportId" });
  const reportID = Number(reportId);
  const report = useReport(reportID);
  const results = useReportResults(reportID);

  if (report.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Report</AlertTitle>
          <AlertDescription>{report.error.message}</AlertDescription>
        </Alert>
      </PageShell>
    );
  }
  if (report.isLoading || !report.data) {
    return (
      <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading...
      </PageShell>
    );
  }

  const rows = reportRows(results.data?.items);
  const resultColumns: ColumnDef<ReportTableRow>[] = resultColumnNames(rows).map((name) => ({
    id: name,
    accessorFn: (row) => row.columns[name] ?? "",
    header: ({ column }) => <DataTableColumnHeader column={column} title={name} />,
    cell: ({ row }) => resultValue(row.original.columns[name]),
  }));
  const columns = [...reportTableColumns({ linkHosts: true }), ...resultColumns];
  return (
    <PageShell>
      <PageHeader
        title={report.data.name}
        description={report.data.description}
        actions={
          <>
            <ShowQueryButton sql={report.data.query} />
            <LiveRunButton to="/osquery/reports/$reportId/live" params={{ reportId }} />
            <EditButton to="/osquery/reports/$reportId/edit" params={{ reportId }}>
              Edit Report
            </EditButton>
          </>
        }
      />

      <div className="grid gap-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold">Results</h2>
          </div>
        </div>
        <DataTable
          columns={columns}
          data={rows}
          isLoading={results.isLoading}
          showExport
          exportFilename={`${report.data.name || "report"}-results.csv`}
          totalCount={rows.length}
          pagination={{ pageIndex: 0, pageSize: rows.length || 50 }}
          sorting={[]}
          clientSort
          onPaginationChange={() => null}
          onSortingChange={() => null}
          empty={
            <Empty>
              <EmptyHeader>
                <EmptyTitle>Nothing to Report Yet</EmptyTitle>
                <EmptyDescription>This report has not stored any result rows.</EmptyDescription>
              </EmptyHeader>
            </Empty>
          }
        />
      </div>
    </PageShell>
  );
}
