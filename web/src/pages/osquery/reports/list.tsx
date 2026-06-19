import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { FileBarChart2, Plus } from "lucide-react";
import * as React from "react";

import { BulkDeleteActionBar } from "@/components/bulk-delete-action-bar";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { selectColumn } from "@/components/data-table/select-column";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/use-auth";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { type OsqueryReport, useBulkDeleteReports, useReports } from "@/hooks/use-reports";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { formatInterval } from "@/lib/utils";

export function ReportListPage() {
  const tableSearch = useDataTableSearch();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";

  const query = useReports({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });

  const reports = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q;

  const columns = React.useMemo<ColumnDef<OsqueryReport>[]>(
    () => reportColumns(isAdmin),
    [isAdmin],
  );

  const table = useDataTable({
    data: reports,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  return (
    <PageShell>
      <PageHeader
        title="Reports"
        actions={
          isAdmin ? (
            <Button asChild size="sm">
              <Link to="/osquery/reports/new">
                <Plus data-icon="inline-start" />
                Create
              </Link>
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
        <DataTableSkeleton columnCount={3} />
      ) : (
        <DataTable
          table={table}
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
              filtered={hasFilters}
              title="No saved queries"
              description="Create a report from SQL."
              filteredDescription="No reports matched the current search."
            />
          }
        >
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput className="h-8 w-40 lg:w-56" />
            </div>
          </div>
        </DataTable>
      )}
    </PageShell>
  );
}

function reportColumns(isAdmin: boolean): ColumnDef<OsqueryReport>[] {
  const columns: ColumnDef<OsqueryReport>[] = [
    selectColumn<OsqueryReport>(),
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
      cell: ({ row }) =>
        isAdmin ? (
          <Link
            to="/osquery/reports/$reportId"
            params={{ reportId: String(row.original.id) }}
            className="font-medium hover:underline"
          >
            {row.original.name}
          </Link>
        ) : (
          <span className="font-medium">{row.original.name}</span>
        ),
      enableHiding: false,
      meta: { label: "Name" },
    },
    {
      id: "schedule_interval",
      accessorKey: "schedule_interval",
      header: ({ column }) => <DataTableColumnHeader column={column} label="Interval" />,
      cell: ({ row }) =>
        row.original.schedule_interval
          ? `Every ${formatInterval(row.original.schedule_interval)}`
          : "Off",
      meta: { label: "Interval" },
    },
  ];
  return isAdmin ? columns : columns.filter((column) => column.id !== "select");
}
