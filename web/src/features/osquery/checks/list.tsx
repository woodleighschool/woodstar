import { getRouteApi } from "@tanstack/react-router";
import { CircleAlert, CircleCheck, MoreHorizontal, Plus, ShieldCheck } from "lucide-react";
import { useState } from "react";

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
import { Link } from "@components/link";
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
import type { OsqueryCheck } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { formatRelative } from "@lib/utils";

import { CheckDeleteDialog } from "./delete-dialog";
import { useBulkDeleteChecks, useChecks } from "./queries";
import { CheckResultCountLink } from "./result-count-link";

const routeApi = getRouteApi("/_authenticated/osquery/checks/");

type CheckTableRow = OsqueryCheck & {
  onDelete: (check: OsqueryCheck) => void;
};

export function CheckListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleting, setDeleting] = useState<OsqueryCheck | null>(null);
  const query = useChecks({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });
  const checks = query.data?.items ?? [];
  const tableRows: CheckTableRow[] = checks.map((check) => ({
    ...check,
    onDelete: setDeleting,
  }));
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: tableRows,
    columns: isAdmin ? checkAdminColumns : checkColumns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
    enableRowSelection: isAdmin,
  });
  return (
    <PageShell>
      <PageHeader
        title="Checks"
        actions={
          isAdmin ? (
            <Button size="sm" render={<Link to="/osquery/checks/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
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
        <DataTableSkeleton columnCount={isAdmin ? 6 : 4} />
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
              filtered={tableSearch.isFiltered}
              title="No health checks"
              description="Create a check from SQL."
              filteredDescription="No checks matched the current search."
            />
          }
        >
          <DataTableSearchInput
            value={tableSearch.q ?? ""}
            onValueChange={tableSearch.onQueryChange}
          />
        </DataTable>
      )}

      {isAdmin ? (
        <CheckDeleteDialog
          open={deleting !== null}
          onOpenChange={(open) => {
            if (!open) setDeleting(null);
          }}
          check={deleting}
        />
      ) : null}
    </PageShell>
  );
}

const checkColumns: DataTableColumnDef<CheckTableRow>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <Link
        to="/osquery/checks/$id"
        params={{ id: String(row.original.id) }}
        className="font-medium"
      >
        {row.original.name}
      </Link>
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
      <CheckResultCountLink
        checkId={row.original.id}
        count={row.original.passing_host_count}
        status="pass"
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
      <CheckResultCountLink
        checkId={row.original.id}
        count={row.original.failing_host_count}
        status="fail"
      />
    ),
    meta: { label: "Fail" },
  },
  {
    id: "updated_at",
    accessorKey: "updated_at",
    header: "Updated",
    cell: ({ row }) => formatRelative(row.original.updated_at),
    meta: { label: "Updated" },
  },
];

function CheckActionsCell({ row }: DataTableCellContext<CheckTableRow>) {
  const check = row.original;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button type="button" size="icon" variant="ghost" />}>
        <MoreHorizontal />
        <span className="sr-only">Open check actions</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem
            render={<Link to="/osquery/checks/$id/edit" params={{ id: String(check.id) }} />}
          >
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onClick={() => check.onDelete(check)}>
            Delete
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

const checkAdminColumns: DataTableColumnDef<CheckTableRow>[] = [
  selectColumn<CheckTableRow>(),
  ...checkColumns,
  {
    id: "actions",
    header: () => null,
    enableSorting: false,
    enableHiding: false,
    size: 44,
    minSize: 44,
    maxSize: 44,
    enableResizing: false,
    cell: CheckActionsCell,
  },
];
