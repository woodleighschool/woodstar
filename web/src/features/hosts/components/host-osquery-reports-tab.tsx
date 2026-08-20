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
import { TextLink } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { listAllHostOsqueryReports, useHostOsqueryReports } from "@features/hosts/queries";
import {
  REPORT_SNAPSHOT_STATUS_OPTIONS,
  ReportResultStatus,
  resultColumnNames,
  resultRowCountLabel,
  serializeSnapshots,
  SnapshotResultRows,
  snapshotStatusLabel,
} from "@features/osquery/reports/query-results";
import { OsqueryResultError } from "@features/osquery/result-error";
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
        <span className="inline-flex w-fit items-center gap-1">
          <TextLink to="/osquery/reports/$id" params={{ id: String(row.original.report_id) }}>
            {row.original.report_name}
          </TextLink>
          {row.original.error ? (
            <OsqueryResultError
              label={`Report error for ${row.original.report_name}`}
              error={row.original.error}
            />
          ) : null}
        </span>
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
    cell: ({ row }) => <ReportResultStatus row={row.original} />,
  },
  {
    id: "reported_at",
    accessorKey: "reported_at",
    header: () => "Last Reported",
    cell: ({ row }) => (row.original.reported_at ? formatRelative(row.original.reported_at) : "-"),
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
      <QueryError title="Failed to Load Reports" error={{ message: "Host route is invalid." }} />
    );
  }

  if (reports.error) {
    return (
      <QueryError
        title="Failed to Load Reports"
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
    { header: "Last Reported", value: (row) => row.reported_at },
    { header: "Error", value: (row) => row.error },
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
          {tableSearch.isFiltered ? "No Matching Reports" : "No Assigned Reports"}
        </PanelEmptyState>
      }
    >
      <DataTableSearchInput
        loading={reports.isPlaceholderData}
        value={tableSearch.q ?? ""}
        onValueChange={tableSearch.onQueryChange}
        placeholder="Search Reports and Results"
      />
      <HostReportsToolbar table={table} />
    </DataTable>
  );
}
