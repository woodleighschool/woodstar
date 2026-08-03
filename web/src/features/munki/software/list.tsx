import { getRouteApi } from "@tanstack/react-router";
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal, PackageSearch, Plus } from "lucide-react";
import * as React from "react";

import { BulkDeleteActionBar } from "@components/bulk-delete-action-bar";
import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { selectColumn } from "@components/data-table/select-column";
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
import { SoftwareArtwork } from "@features/software/software-icon";
import type { MunkiSoftware } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { formatRelative } from "@lib/utils";

import { MunkiSoftwareDeleteDialog } from "./delete-dialog";
import { useBulkDeleteMunkiSoftware, useMunkiSoftware } from "./queries";

const routeApi = getRouteApi("/_authenticated/munki/software/");

function SoftwareNameCell({ row }: CellContext<MunkiSoftware, unknown>) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <SoftwareArtwork src={row.original.icon_url} />
      <Link
        to="/munki/software/$id"
        params={{ id: String(row.original.id) }}
        className="min-w-0 truncate font-medium"
        title={row.original.name}
      >
        {row.original.name}
      </Link>
    </div>
  );
}

function softwareColumns(
  isAdmin: boolean,
  onDelete: (software: MunkiSoftware) => void,
): ColumnDef<MunkiSoftware>[] {
  const columns: ColumnDef<MunkiSoftware>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: "Name",
      cell: SoftwareNameCell,
      enableHiding: false,
      size: 340,
      minSize: 180,
      meta: { label: "Name" },
    },
    {
      id: "category",
      accessorKey: "category",
      header: "Category",
      cell: ({ row }) => row.original.category || "-",
      size: 144,
      meta: { label: "Category" },
    },
    {
      id: "developer",
      accessorKey: "developer",
      header: "Developer",
      cell: ({ row }) => row.original.developer || "-",
      size: 200,
      meta: { label: "Developer" },
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: "Updated",
      cell: ({ row }) => formatRelative(row.original.updated_at),
      size: 144,
      meta: { label: "Updated" },
    },
  ];
  if (!isAdmin) return columns;
  return [
    selectColumn<MunkiSoftware>(),
    ...columns,
    {
      id: "actions",
      header: () => null,
      enableSorting: false,
      enableHiding: false,
      size: 44,
      minSize: 44,
      maxSize: 44,
      enableResizing: false,
      cell: ({ row }) => (
        <SoftwareRowActions software={row.original} onDelete={() => onDelete(row.original)} />
      ),
    },
  ];
}

export function MunkiSoftwareListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleting, setDeleting] = React.useState<MunkiSoftware | null>(null);
  const query = useMunkiSoftware({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });
  const software = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const columns = React.useMemo(() => softwareColumns(isAdmin, setDeleting), [isAdmin]);
  const table = useDataTable({
    tableState: tableSearch,
    data: software,
    columns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
    enableRowSelection: isAdmin,
  });
  return (
    <PageShell>
      <PageHeader
        title="Software"
        actions={
          isAdmin ? (
            <Button size="sm" render={<Link to="/munki/software/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />

      {query.error ? (
        <QueryError
          title="Failed to load software"
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
              <BulkDeleteActionBar
                table={table}
                useBulkDelete={useBulkDeleteMunkiSoftware}
                noun="software"
                pluralNoun="software"
                description="Packages and targeting for the selected software will also be removed."
              />
            ) : undefined
          }
          empty={
            <DataTableEmpty
              icon={<PackageSearch />}
              filtered={tableSearch.isFiltered}
              filteredTitle="No matching software"
              title="No software"
              description="Create software to manage Munki packages."
              filteredDescription="Try a different search."
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
        <MunkiSoftwareDeleteDialog
          software={deleting}
          open={deleting !== null}
          onOpenChange={(open) => {
            if (!open) setDeleting(null);
          }}
        />
      ) : null}
    </PageShell>
  );
}

function SoftwareRowActions({
  software,
  onDelete,
}: {
  software: MunkiSoftware;
  onDelete: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button type="button" size="icon" variant="ghost" />}>
        <MoreHorizontal />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem
            render={<Link to="/munki/software/$id/edit" params={{ id: String(software.id) }} />}
          >
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onClick={onDelete}>
            Delete
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
