import { useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2 } from "lucide-react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import {
  EditButton,
  ExportButton,
  LiveRunButton,
  ShowQueryButton,
} from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { useQueryDetail, useQueryResults } from "@/hooks/use-queries";
import {
  reportRows,
  reportTableColumns,
  resultColumnNames,
  resultValue,
  type ReportTableRow,
} from "@/lib/query-results";

export function ReportDetailPage() {
  const { reportId } = useParams({ from: "/_authenticated/reports/$reportId" });
  const query = useQueryDetail(reportId);
  const results = useQueryResults(reportId);

  if (query.error) {
    return (
      <div className="p-6">
        <Alert variant="destructive">
          <AlertTitle>Failed to load report</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      </div>
    );
  }
  if (query.isLoading || !query.data) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading...
      </div>
    );
  }

  const rows = reportRows(results.data?.items);
  const resultColumns: ColumnDef<ReportTableRow>[] = resultColumnNames(rows).map((name) => ({
    id: name,
    accessorFn: (row) => row.columns[name] ?? "",
    header: ({ column }) => <DataTableColumnHeader column={column} title={name} />,
    cell: ({ row }) => (
      <span className="block max-w-[28rem] truncate" title={row.original.columns[name] ?? ""}>
        {resultValue(row.original.columns[name])}
      </span>
    ),
  }));
  const columns = [...reportTableColumns({ linkHosts: true }), ...resultColumns];
  const hasResults = rows.length > 0;

  return (
    <div className="flex flex-col gap-5 p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-2xl font-semibold tracking-tight">{query.data.name}</h1>
          {query.data.description ? (
            <p className="text-muted-foreground mt-1 max-w-3xl text-sm">{query.data.description}</p>
          ) : null}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <ShowQueryButton sql={query.data.query} />
          <LiveRunButton to="/reports/$reportId/live" params={{ reportId }} />
          <EditButton to="/reports/$reportId/edit" params={{ reportId }}>
            Edit report
          </EditButton>
        </div>
      </div>

      <div className="grid gap-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold">Results</h2>
            <p className="text-muted-foreground text-sm">
              Stored snapshot results by host, flattened into query columns.
            </p>
          </div>
          <ExportButton disabled={!hasResults} onClick={() => exportReport(query.data.name, rows)} />
        </div>
        <DataTable
          columns={columns}
          data={rows}
          isLoading={results.isLoading}
          totalCount={rows.length}
          page={1}
          perPage={rows.length}
          sort={{}}
          clientSort
          onPageChange={() => null}
          onPerPageChange={() => null}
          onSortChange={() => null}
          empty={
            <Empty>
              <EmptyHeader>
                <EmptyTitle>Nothing to report yet</EmptyTitle>
                <EmptyDescription>This report has not stored any result rows.</EmptyDescription>
              </EmptyHeader>
            </Empty>
          }
        />
      </div>
    </div>
  );
}

function exportReport(name: string, rows: ReportTableRow[]) {
  const dynamicColumns = resultColumnNames(rows);
  const headers = ["host_id", "host_display_name", "last_fetched", ...dynamicColumns];
  const csv = [
    headers.join(","),
    ...rows.map((row) =>
      [
        String(row.hostId),
        row.hostName,
        row.lastFetched ?? "",
        ...dynamicColumns.map((column) => row.columns[column] ?? ""),
      ]
        .map(csvCell)
        .join(","),
    ),
  ].join("\n");

  const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `${name || "Report"} - Report.csv`;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function csvCell(value: string) {
  return `"${value.replaceAll('"', '""')}"`;
}
