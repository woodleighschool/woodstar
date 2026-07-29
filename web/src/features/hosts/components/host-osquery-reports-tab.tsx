import type { ColumnDef } from "@tanstack/react-table";

import { DataTableClient } from "@components/data-table/data-table-client";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableRowExpander } from "@components/data-table/data-table-row-expander";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { Skeleton } from "@components/ui/skeleton";
import { useHostOsqueryReports } from "@features/hosts/queries";
import {
  reportSnapshotRows,
  type ReportSnapshotTableRow,
  resultColumnNames,
  serializeSnapshots,
  SnapshotResultRows,
  snapshotSearchText,
  snapshotStatusLabel,
  SnapshotStatusBadge,
} from "@features/osquery/reports/query-results";
import { formatRelative } from "@lib/utils";

const hostReportColumns: ColumnDef<ReportSnapshotTableRow>[] = [
  {
    id: "expand",
    header: () => <span className="sr-only">Expand</span>,
    cell: ({ row }) => <DataTableRowExpander row={row} label={row.original.reportName} />,
    enableSorting: false,
    size: 44,
  },
  {
    id: "reportName",
    accessorKey: "reportName",
    header: () => "Report",
    cell: ({ row }) => (
      <div className="flex flex-col gap-0.5">
        <Link
          to="/osquery/reports/$id"
          params={{ id: String(row.original.reportId) }}
          className="w-fit"
        >
          {row.original.reportName}
        </Link>
        {row.original.reportDescription ? (
          <span className="max-w-xl truncate text-xs text-muted-foreground">
            {row.original.reportDescription}
          </span>
        ) : null}
      </div>
    ),
  },
  {
    id: "status",
    accessorFn: snapshotStatusLabel,
    header: () => "Collection",
    cell: ({ row }) => <SnapshotStatusBadge row={row.original} />,
  },
  {
    id: "collectedAt",
    accessorKey: "collectedAt",
    header: () => "Collected",
    cell: ({ row }) => (row.original.collectedAt ? formatRelative(row.original.collectedAt) : "-"),
  },
  {
    id: "rowCount",
    accessorFn: (row) => row.rows.length,
    header: () => "Result Rows",
  },
];

export function HostOsqueryReportsTab({ hostId }: { hostId: number | null }) {
  const reports = useHostOsqueryReports(hostId);

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
    return <Skeleton className="h-64 w-full" />;
  }

  const rows = reportSnapshotRows(reports.data);
  const columnNames = resultColumnNames(rows.flatMap((row) => row.rows));
  const exportMetadata: DataTableExportOptions<ReportSnapshotTableRow>["columns"] = [
    { header: "Report", value: (row) => row.reportName },
    { header: "Collection Status", value: snapshotStatusLabel },
    { header: "Collected At", value: (row) => row.collectedAt },
  ];
  const exportOptions: DataTableExportOptions<ReportSnapshotTableRow> = {
    filename: `host-${hostId}-osquery-reports`,
    columns: exportMetadata,
    serializeRows: (exportRows) => serializeSnapshots(exportRows, exportMetadata, columnNames),
  };

  return (
    <DataTableClient
      title="Reports"
      columns={hostReportColumns}
      data={rows}
      exportOptions={exportOptions}
      getRowCanExpand={(row) => row.original.rows.length > 0}
      getRowId={(row) => row.id}
      getSearchText={snapshotSearchText}
      initialSorting={[{ id: "reportName", desc: false }]}
      renderSubRow={(row) => <SnapshotResultRows rows={row.original.rows} />}
      searchPlaceholder="Search reports and results"
      empty={<PanelEmptyState>No assigned reports</PanelEmptyState>}
    />
  );
}
