import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2 } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/ui/data-table";
import { DataTableColumnHeader } from "@/components/ui/data-table-column-header";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { useHostQueryResults } from "@/hooks/use-hosts";
import { useQueryDetail } from "@/hooks/use-queries";
import { reportRows, resultColumnNames, resultValue, type ReportTableRow } from "@/lib/query-results";
import { formatRelative } from "@/lib/utils";

export function HostReportResultsPage() {
  const { hostId, reportId } = useParams({ from: "/_authenticated/hosts/$hostId/reports/$reportId" });
  const report = useQueryDetail(reportId);
  const results = useHostQueryResults(hostId, reportId);

  if (report.error || results.error) {
    return (
      <div className="p-6">
        <Alert variant="destructive">
          <AlertTitle>Failed to load report results</AlertTitle>
          <AlertDescription>{report.error?.message ?? results.error?.message}</AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!report.data || results.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading report results...
      </div>
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
    <div className="flex flex-col gap-4 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{report.data.name}</h1>
          <p className="text-muted-foreground text-sm">
            {results.data?.host_name ?? "Host"}
            {results.data?.last_fetched ? ` · last fetched ${formatRelative(results.data.last_fetched)}` : ""}
          </p>
        </div>
        <Button asChild size="sm" variant="outline">
          <Link to="/reports/$reportId" params={{ reportId }}>
            View all hosts
          </Link>
        </Button>
      </div>

      <DataTable
        columns={columns}
        data={rows}
        totalCount={rows.length}
        page={1}
        perPage={Math.max(rows.length, 50)}
        sort={{}}
        onPageChange={() => null}
        onPerPageChange={() => null}
        onSortChange={() => null}
        empty={
          <Empty>
            <EmptyHeader>
              <EmptyTitle>{results.data?.last_fetched ? "Nothing to report" : "Collecting results"}</EmptyTitle>
              <EmptyDescription>
                {results.data?.last_fetched
                  ? "This report ran on the host but returned no rows."
                  : "This host has not sent report results yet."}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        }
      />
    </div>
  );
}
