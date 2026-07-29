import { getRouteApi, useParams } from "@tanstack/react-router";
import { getExpandedRowModel, type ColumnDef, type Table } from "@tanstack/react-table";
import { Pencil, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTable } from "@components/data-table/data-table";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableRowExpander } from "@components/data-table/data-table-row-expander";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { QueryGate } from "@components/query-gate";
import { LabelTargetDetails } from "@components/targeting/target-details";
import { Button } from "@components/ui/button";
import { Skeleton } from "@components/ui/skeleton";
import { useAuth } from "@features/auth/queries";
import { LiveRunButton, ShowQueryButton } from "@features/osquery/live/query-actions";
import type { OsqueryReportSnapshot } from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { formatInterval, formatRelative } from "@lib/utils";

import { ReportDeleteDialog } from "./delete-dialog";
import { listAllReportSnapshots, useReport, useReportSnapshots } from "./queries";
import {
  parseReportSnapshotStatus,
  REPORT_SNAPSHOT_STATUS_OPTIONS,
  resultColumnNames,
  resultRowCountLabel,
  serializeSnapshots,
  SnapshotResultRows,
  snapshotStatusLabel,
  SnapshotStatusBadge,
} from "./query-results";

const EMPTY_REPORT_SNAPSHOTS: OsqueryReportSnapshot[] = [];

const reportSnapshotColumns: ColumnDef<OsqueryReportSnapshot>[] = [
  {
    id: "expand",
    header: () => <span className="sr-only">Expand</span>,
    cell: ({ row }) => <DataTableRowExpander row={row} label={row.original.host_name} />,
    enableSorting: false,
    size: 44,
  },
  {
    id: "host_name",
    accessorKey: "host_name",
    header: () => "Host",
    cell: ({ row }) => (
      <Link to="/hosts/$id" params={{ id: String(row.original.host_id) }}>
        {row.original.host_name}
      </Link>
    ),
  },
  {
    id: "status",
    accessorKey: "status",
    header: () => "Status",
    enableColumnFilter: true,
    cell: ({ row }) => <SnapshotStatusBadge row={row.original} />,
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
const routeApi = getRouteApi("/_authenticated/osquery/reports/$id/");

function ReportResultsToolbar({ table }: { table: Table<OsqueryReportSnapshot> }) {
  return (
    <DataTableFacetedFilter
      column={table.getColumn("status")}
      title="Status"
      options={REPORT_SNAPSHOT_STATUS_OPTIONS}
    />
  );
}

export function ReportDetailPage() {
  const { id: reportId } = useParams({
    from: "/_authenticated/osquery/reports/$id",
  });
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: STATUS_FILTER_KEYS,
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleteOpen, setDeleteOpen] = useState(false);
  const id = parseRouteID(reportId);
  const report = useReport(id);
  const status = parseReportSnapshotStatus(tableSearch.filters.status?.[0]);
  const snapshots = useReportSnapshots(id, {
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    status,
  });
  const rows = snapshots.data?.items ?? EMPTY_REPORT_SNAPSHOTS;
  const totalCount = snapshots.data?.count ?? 0;
  const pageCount = snapshots.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const columnNames = useMemo(() => resultColumnNames(rows.flatMap((row) => row.rows)), [rows]);
  const table = useDataTable({
    tableState: tableSearch,
    data: rows,
    columns: reportSnapshotColumns,
    pageCount,
    rowCount: totalCount,
    getRowId: (row) => `${row.report_id}-${row.host_id}`,
    getRowCanExpand: (row) => row.original.rows.length > 0,
    getExpandedRowModel: getExpandedRowModel(),
    paginateExpandedRows: false,
  });

  if (id === null) {
    return (
      <QueryGate title="Failed to load report" error={{ message: "Report route is invalid." }} />
    );
  }

  if (report.error) {
    return (
      <QueryGate
        title="Failed to load report"
        error={report.error}
        onRetry={() => void report.refetch()}
      />
    );
  }

  if (!report.data) {
    return (
      <PageShell>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </PageShell>
    );
  }

  const exportMetadata: DataTableExportOptions<OsqueryReportSnapshot>["columns"] = [
    { header: "Host", value: (row) => row.host_name },
    { header: "Status", value: snapshotStatusLabel },
    { header: "Last Collected", value: (row) => row.collected_at },
  ];
  const exportOptions: DataTableExportOptions<OsqueryReportSnapshot> = {
    filename: `osquery-report-${id}-results`,
    columns: exportMetadata,
    loadRows: () =>
      listAllReportSnapshots(id, {
        q: tableSearch.q,
        sort: tableSearch.sort,
        status,
      }),
    serializeRows: (exportRows) => serializeSnapshots(exportRows, exportMetadata),
  };

  return (
    <PageShell>
      <PageHeader
        title="Report Details"
        description={report.data.description || undefined}
        meta={`Edited ${formatRelative(report.data.updated_at)}`}
        actions={
          <>
            {isAdmin ? (
              <>
                <Button
                  size="sm"
                  render={<Link to="/osquery/reports/$id/edit" params={{ id: reportId }} />}
                  nativeButton={false}
                >
                  <Pencil data-icon="inline-start" />
                  Edit
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  size="sm"
                  onClick={() => setDeleteOpen(true)}
                >
                  <Trash2 data-icon="inline-start" />
                  Delete
                </Button>
              </>
            ) : null}
            <ShowQueryButton sql={report.data.query} />
            <LiveRunButton kind="report" id={id} sql={report.data.query} />
          </>
        }
      />

      <KeyValueSection title="Overview">
        <KeyValueRow label="Name" value={report.data.name} />
        <KeyValueRow
          label="Interval"
          value={
            report.data.schedule_interval
              ? `Every ${formatInterval(report.data.schedule_interval)}`
              : "Off"
          }
        />
        <KeyValueRow label="Minimum Osquery" value={report.data.min_osquery_version || "Any"} />
      </KeyValueSection>

      <LabelTargetDetails targets={report.data.targets} />

      {snapshots.error ? (
        <QueryError
          title="Failed to load report results"
          error={snapshots.error}
          onRetry={() => void snapshots.refetch()}
        />
      ) : snapshots.isLoading ? (
        <DataTableSkeleton columnCount={5} filterCount={1} withExport withViewOptions={false} />
      ) : (
        <DataTable
          table={table}
          heading="Results"
          exportOptions={exportOptions}
          renderSubRow={(row) => (
            <SnapshotResultRows rows={row.original.rows} columnNames={columnNames} />
          )}
          empty={
            <PanelEmptyState>
              {tableSearch.isFiltered ? "No matching report results" : "No targeted hosts"}
            </PanelEmptyState>
          }
        >
          <div className="flex flex-wrap items-center gap-2">
            <DataTableSearchInput
              value={tableSearch.q ?? ""}
              onValueChange={tableSearch.onQueryChange}
              placeholder="Search hosts and results"
              className="h-8 w-full sm:w-64"
            />
            <ReportResultsToolbar table={table} />
          </div>
        </DataTable>
      )}

      {isAdmin ? (
        <ReportDeleteDialog
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
          report={report.data}
          onDeleted={() => void navigate({ to: "/osquery/reports" })}
        />
      ) : null}
    </PageShell>
  );
}
