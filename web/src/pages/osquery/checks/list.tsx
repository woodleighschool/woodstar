import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { CircleAlert, CircleCheck, Plus, ShieldCheck } from "lucide-react";
import * as React from "react";

import { BulkDeleteActionBar } from "@/components/bulk-delete-action-bar";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { selectColumn } from "@/components/data-table/select-column";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { useAuth } from "@/hooks/use-auth";
import { type OsqueryCheck, useBulkDeleteChecks, useChecks } from "@/hooks/use-checks";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";

export function CheckListPage() {
  const tableSearch = useDataTableSearch();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";

  const query = useChecks({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });

  const checks = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q;

  const columns = React.useMemo<ColumnDef<OsqueryCheck>[]>(() => checkColumns(isAdmin), [isAdmin]);

  const table = useDataTable({
    data: checks,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  return (
    <PageShell>
      <PageHeader
        title="Checks"
        actions={
          isAdmin ? (
            <Button asChild size="sm">
              <Link to="/osquery/checks/new">
                <Plus data-icon="inline-start" />
                Create
              </Link>
            </Button>
          ) : null
        }
      />
      {query.error ? (
        <QueryError
          title="Failed to load checks"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={4} />
      ) : (
        <DataTable
          table={table}
          actionBar={
            isAdmin ? (
              <BulkDeleteActionBar table={table} useBulkDelete={useBulkDeleteChecks} noun="check" />
            ) : undefined
          }
          empty={
            <DataTableEmpty
              icon={<ShieldCheck />}
              filtered={hasFilters}
              title="No health checks"
              description="Create a check from SQL."
              filteredDescription="No checks matched the current search."
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

function checkColumns(isAdmin: boolean): ColumnDef<OsqueryCheck>[] {
  const columns: ColumnDef<OsqueryCheck>[] = [
    selectColumn<OsqueryCheck>(),
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
      cell: ({ row }) =>
        isAdmin ? (
          <Link
            to="/osquery/checks/$checkId"
            params={{ checkId: String(row.original.id) }}
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
      id: "passing_host_count",
      accessorKey: "passing_host_count",
      enableSorting: false,
      header: () => (
        <span className="flex items-center gap-1.5">
          <CircleCheck className="size-4 text-status-online" />
          Pass
        </span>
      ),
      cell: ({ row }) => (
        <HostCount
          checkId={row.original.id}
          response="pass"
          value={row.original.passing_host_count}
        />
      ),
      meta: { label: "Pass" },
    },
    {
      id: "failing_host_count",
      accessorKey: "failing_host_count",
      enableSorting: false,
      header: () => (
        <span className="flex items-center gap-1.5">
          <CircleAlert className="size-4 text-muted-foreground" />
          Fail
        </span>
      ),
      cell: ({ row }) => (
        <HostCount
          checkId={row.original.id}
          response="fail"
          value={row.original.failing_host_count}
        />
      ),
      meta: { label: "Fail" },
    },
  ];
  return isAdmin ? columns : columns.filter((column) => column.id !== "select");
}

function HostCount({
  checkId,
  response,
  value,
}: {
  checkId: number;
  response: "pass" | "fail";
  value: number;
}) {
  return (
    <Link
      to="/hosts"
      search={{ check_id: checkId, check_response: response }}
      className="hover:underline"
    >
      {value} {value === 1 ? "host" : "hosts"}
    </Link>
  );
}
