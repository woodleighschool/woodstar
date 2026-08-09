import { getRouteApi } from "@tanstack/react-router";
import { useMemo } from "react";

import { DataTable } from "@components/data-table/data-table";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableRowExpander } from "@components/data-table/data-table-row-expander";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableColumnDef, DataTableInstance } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { listAllHostOsqueryReports, useHostOsqueryReports } from "@features/hosts/queries";
import {
  REPORT_SNAPSHOT_STATUS_OPTIONS,
  resultColumnNames,
  resultRowCountLabel,
  serializeSnapshots,
  SnapshotResultRows,
  snapshotStatusLabel,
} from "@features/osquery/reports/query-results";
import type { OsqueryReportSnapshot } from "@lib/api";
import { formatRelative } from "@lib/utils";

const EMPTY_REPORT_SNAPSHOTS: OsqueryReportSnapshot[] = [];

const hostReportColumns: DataTableColumnDef<OsqueryReportSnapshot>[] = [
  {
    id: "expand",
    header: () => <span className="sr-only">Expand</span>,
    cell: ({ row }) => <DataTableRowExpander row={row} label={row.original.report_name} />,
    enableSorting: false,
    size: 44,
    minSize: 44,
    maxSize: 44,
    enableResizing: false,
  },
  {
    id: "report_name",
    accessorKey: "report_name",
    header: () => "Report",
    cell: ({ row }) => (
      <div className="flex flex-col gap-0.5">
        <Link
          to="/osquery/reports/$id"
          params={{ id: String(row.original.report_id) }}
          className="w-fit"
        >
          {row.original.report_name}
        </Link>
        {row.original.report_description ? (
          <span className="text-xs whitespace-normal text-muted-foreground">
            {row.original.report_description}
          </span>
        ) : null}
      </div>
    ),
  },
  {
    id: "status",
    accessorKey: "status",
    header: () => "Status",
    enableColumnFilter: true,
    cell: ({ row }) => snapshotStatusLabel(row.original),
  },
  {
    id: "collected_at",
    accessorKey: "collected_at",
    header: () => "Last Collected",
    cell: ({ row }) =>
      row.original.collected_at ? formatRelative(row.original.collected_at) : "-",
  },
  {
    id: "result_row_count",
    accessorKey: "result_row_count",
    header: () => "Result Rows",
    cell: ({ row }) => resultRowCountLabel(row.original),
  },
];

const STATUS_FILTER_KEYS = [{ id: "status" }] as const;
const routeApi = getRouteApi("/_authenticated/hosts/$id/reports");

function HostReportsToolbar({ table }: { table: DataTableInstance<OsqueryReportSnapshot> }) {
  return (
    <DataTableFacetedFilter
      column={table.getColumn("status")}
      title="Status"
      options={REPORT_SNAPSHOT_STATUS_OPTIONS}
    />
  );
}

export function HostOsqueryReportsTab({ hostId }: { hostId: number | null }) {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: STATUS_FILTER_KEYS,
  });
  const status = search.status;
  const reports = useHostOsqueryReports(hostId, {
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    status,
  });
  const rows = reports.data?.items ?? EMPTY_REPORT_SNAPSHOTS;
  const totalCount = reports.data?.count ?? 0;
  const pageCount = reports.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const columnNames = useMemo(() => resultColumnNames(rows.flatMap((row) => row.rows)), [rows]);
  const table = useDataTable({
    tableState: tableSearch,
    data: rows,
    columns: hostReportColumns,
    pageCount,
    rowCount: totalCount,
    getRowId: (row) => `${row.report_id}-${row.host_id}`,
    getRowCanExpand: (row) => row.original.rows.length > 0,
    paginateExpandedRows: false,
  });

  if (hostId === null) {
    return (
      <QueryError title="Failed to load reports" error={{ message: "Host route is invalid." }} />
    );
  }

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
    return <DataTableSkeleton columnCount={5} filterCount={1} withExport />;
  }

  const exportMetadata: DataTableExportOptions<OsqueryReportSnapshot>["columns"] = [
    { header: "Report", value: (row) => row.report_name },
    { header: "Status", value: snapshotStatusLabel },
    { header: "Last Collected", value: (row) => row.collected_at },
  ];
  const exportOptions: DataTableExportOptions<OsqueryReportSnapshot> = {
    filename: `host-${hostId}-osquery-reports`,
    columns: exportMetadata,
    loadRows: () =>
      listAllHostOsqueryReports(hostId, {
        q: tableSearch.q,
        sort: tableSearch.sort,
        status,
      }),
    serializeRows: (exportRows) => serializeSnapshots(exportRows, exportMetadata),
  };

  return (
    <DataTable
      table={table}
      pending={reports.isPlaceholderData}
      heading="Reports"
      exportOptions={exportOptions}
      renderSubRow={(row) => (
        <SnapshotResultRows rows={row.original.rows} columnNames={columnNames} />
      )}
      empty={
        <PanelEmptyState>
          {tableSearch.isFiltered ? "No matching reports" : "No assigned reports"}
        </PanelEmptyState>
      }
    >
      <DataTableSearchInput
        loading={reports.isPlaceholderData}
        value={tableSearch.q ?? ""}
        onValueChange={tableSearch.onQueryChange}
        placeholder="Search reports and results"
      />
      <HostReportsToolbar table={table} />
    </DataTable>
  );
}
