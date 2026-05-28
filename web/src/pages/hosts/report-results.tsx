import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2 } from "lucide-react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { useHostReportResults } from "@/hooks/use-hosts";
import { useReport } from "@/hooks/use-reports";
import { reportRows, resultColumnNames, resultValue, type ReportTableRow } from "@/lib/query-results";
import { formatRelative } from "@/lib/utils";

export function HostReportResultsPage() {
  const { hostId, reportId } = useParams({ from: "/_authenticated/hosts/$hostId/reports/$reportId" });
  const report = useReport(Number(reportId));
  const results = useHostReportResults(Number(hostId), Number(reportId));

  if (report.error || results.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Report Results</AlertTitle>
          <AlertDescription>{report.error?.message ?? results.error?.message}</AlertDescription>
        </Alert>
      </PageShell>
    );
  }

  if (!report.data || results.isLoading) {
    return (
      <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading Report Results...
      </PageShell>
    );
  }

  const rows = reportRows(results.data?.items);
  const resultColumns: ColumnDef<ReportTableRow>[] = resultColumnNames(rows).map((name) => ({
    id: name,
    accessorFn: (row) => row.columns[name] ?? "",
    header: ({ column }) => <DataTableColumnHeader column={column} title={name} />,
    cell: ({ row }) => (
      <span className="block max-w-[32rem] truncate" title={row.original.columns[name] ?? ""}>
        {resultValue(row.original.columns[name])}
      </span>
    ),
  }));
  const columns =
    resultColumns.length > 0
      ? resultColumns
      : [
          {
            id: "results",
            header: "Results",
            cell: () => null,
          } satisfies ColumnDef<ReportTableRow>,
        ];

  return (
    <PageShell>
      <PageHeader
        title={report.data.name}
        description={
          <>
            {results.data?.host_name ?? "Host"}
            {results.data?.last_fetched ? ` · last fetched ${formatRelative(results.data.last_fetched)}` : ""}
          </>
        }
        actions={
          <Button asChild size="sm" variant="outline">
            <Link to="/osquery/reports/$reportId" params={{ reportId }}>
              View all hosts
            </Link>
          </Button>
        }
      />

      <DataTable
        columns={columns}
        data={rows}
        totalCount={rows.length}
        pagination={{ pageIndex: 0, pageSize: rows.length || 50 }}
        sorting={[]}
        clientSort
        onPaginationChange={() => null}
        onSortingChange={() => null}
        empty={
          <Empty>
            <EmptyHeader>
              <EmptyTitle>{results.data?.last_fetched ? "Nothing to Report" : "Collecting Results"}</EmptyTitle>
              <EmptyDescription>
                {results.data?.last_fetched
                  ? "This report ran on the host but returned no rows."
                  : "This host has not sent report results yet."}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        }
      />
    </PageShell>
  );
}
