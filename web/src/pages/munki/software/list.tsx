import { getRouteApi, Link } from "@tanstack/react-router";
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import { PackageSearch, Plus } from "lucide-react";

import { BulkDeleteActionBar } from "@/components/bulk-delete-action-bar";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { selectColumn } from "@/components/data-table/select-column";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { SoftwareArtwork } from "@/components/software/software-icon";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/use-auth";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { useBulkDeleteMunkiSoftware, useMunkiSoftware } from "@/hooks/use-munki-software";
import type { MunkiSoftware } from "@/lib/api";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";

const routeApi = getRouteApi("/_authenticated/munki/software/");

function SoftwareNameCell({ row }: CellContext<MunkiSoftware, unknown>) {
  const { user } = useAuth();
  const content = (
    <>
      <SoftwareArtwork src={row.original.icon_url} />
      <span className="truncate">{row.original.name}</span>
    </>
  );
  return user?.role === "admin" ? (
    <Link
      to="/munki/software/$id"
      params={{ id: String(row.original.id) }}
      className="flex min-w-0 items-center gap-2 font-medium hover:underline"
    >
      {content}
    </Link>
  ) : (
    <span className="flex min-w-0 items-center gap-2 font-medium">{content}</span>
  );
}

const softwareColumns: ColumnDef<MunkiSoftware>[] = [
  selectColumn<MunkiSoftware>(),
  {
    id: "name",
    accessorKey: "name",
    header: "Name",
    cell: SoftwareNameCell,
    enableHiding: false,
    meta: { label: "Name" },
  },
  {
    id: "category",
    accessorKey: "category",
    header: "Category",
    cell: ({ row }) => row.original.category || "-",
    meta: { label: "Category" },
  },
  {
    id: "developer",
    accessorKey: "developer",
    header: "Developer",
    cell: ({ row }) => row.original.developer || "-",
    meta: { label: "Developer" },
  },
  {
    id: "updated_at",
    accessorKey: "updated_at",
    header: "Updated",
    cell: ({ row }) => formatRelative(row.original.updated_at),
    meta: { label: "Updated" },
  },
];

export function MunkiSoftwareListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const query = useMunkiSoftware({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });
  const software = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: software,
    columns: softwareColumns,
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
        <DataTableSkeleton columnCount={5} />
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
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput
                className="h-8 w-40 lg:w-56"
                value={tableSearch.q ?? ""}
                onValueChange={tableSearch.onQueryChange}
              />
            </div>
          </div>
        </DataTable>
      )}
    </PageShell>
  );
}
