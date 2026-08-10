import { getRouteApi } from "@tanstack/react-router";
import { FileBarChart2, MoreHorizontal, Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { BulkDeleteActionBar } from "@components/bulk-delete-action-bar";
import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { selectColumn } from "@components/data-table/select-column";
import type { DataTableCellContext, DataTableColumnDef } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@components/ui/dropdown-menu";
import { useAuth } from "@features/auth/queries";
import type { OsqueryReport } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { formatInterval, formatRelative } from "@lib/utils";

import { ReportDeleteDialog } from "./delete-dialog";
import { useBulkDeleteReports, useReports } from "./queries";

const routeApi = getRouteApi("/_authenticated/osquery/reports/");

type ReportTableRow = OsqueryReport & {
  onDelete: (report: OsqueryReport) => void;
};

export function ReportListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleting, setDeleting] = useState<OsqueryReport | null>(null);
  const query = useReports({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });
  const tableRows = useMemo<ReportTableRow[]>(
    () => query.data?.items.map((report) => ({ ...report, onDelete: setDeleting })) ?? [],
    [query.data?.items],
  );
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: tableRows,
    columns: isAdmin ? reportAdminColumns : reportColumns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
    enableRowSelection: isAdmin,
  });
  return (
    <PageShell>
      <PageHeader
        title="Reports"
        actions={
          isAdmin ? (
            <Button size="sm" render={<Link to="/osquery/reports/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />
      {query.error ? (
        <QueryError
          title="Failed to load reports"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={isAdmin ? 5 : 3} />
      ) : (
        <DataTable
          table={table}
          pending={query.isPlaceholderData}
          actionBar={
            isAdmin ? (
              <BulkDeleteActionBar
                table={table}
                useBulkDelete={useBulkDeleteReports}
                noun="report"
              />
            ) : undefined
          }
          empty={
            <DataTableEmpty
              icon={<FileBarChart2 />}
              filtered={tableSearch.isFiltered}
              title="No saved queries"
              description="Create a report from SQL."
              filteredDescription="No reports matched the current search."
            />
          }
        >
          <DataTableSearchInput
            loading={query.isPlaceholderData}
            value={tableSearch.q ?? ""}
            onValueChange={tableSearch.onQueryChange}
          />
        </DataTable>
      )}

      {isAdmin ? (
        <ReportDeleteDialog
          open={deleting !== null}
          onOpenChange={(open) => {
            if (!open) setDeleting(null);
          }}
          report={deleting}
        />
      ) : null}
    </PageShell>
  );
}

const reportColumns: DataTableColumnDef<ReportTableRow>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <TextLink
        to="/osquery/reports/$id"
        params={{ id: String(row.original.id) }}
        className="font-medium"
      >
        {row.original.name}
      </TextLink>
    ),
    enableHiding: false,
    meta: { label: "Name" },
  },
  {
    id: "schedule_interval",
    accessorKey: "schedule_interval",
    header: "Interval",
    cell: ({ row }) =>
      row.original.schedule_interval
        ? `Every ${formatInterval(row.original.schedule_interval)}`
        : "Off",
    meta: { label: "Interval" },
  },
  {
    id: "updated_at",
    accessorKey: "updated_at",
    header: "Updated",
    cell: ({ row }) => formatRelative(row.original.updated_at),
    meta: { label: "Updated" },
  },
];

function ReportActionsCell({ row }: DataTableCellContext<ReportTableRow>) {
  const report = row.original;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button type="button" size="icon" variant="ghost" />}>
        <MoreHorizontal />
        <span className="sr-only">Open report actions</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem
            render={<Link to="/osquery/reports/$id/edit" params={{ id: String(report.id) }} />}
          >
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onClick={() => report.onDelete(report)}>
            Delete
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

const reportAdminColumns: DataTableColumnDef<ReportTableRow>[] = [
  selectColumn<ReportTableRow>(),
  ...reportColumns,
  {
    id: "actions",
    header: () => null,
    enableSorting: false,
    enableHiding: false,
    size: 44,
    minSize: 44,
    maxSize: 44,
    enableResizing: false,
    cell: ReportActionsCell,
  },
];
