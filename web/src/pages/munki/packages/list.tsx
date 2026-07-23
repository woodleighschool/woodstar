import { Link } from "@tanstack/react-router";
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import { PackageCheck, Plus } from "lucide-react";

import { BulkDeleteActionBar } from "@/components/bulk-delete-action-bar";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { selectColumn } from "@/components/data-table/select-column";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { SoftwareArtwork } from "@/components/software/software-icon";
import { Button } from "@/components/ui/button";
import { formatBytes } from "@/components/ui/file-upload";
import { useAuth } from "@/hooks/use-auth";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { useBulkDeleteMunkiPackages, useMunkiPackages } from "@/hooks/use-munki-packages";
import type { MunkiPackage } from "@/lib/api";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative, isOneOf } from "@/lib/utils";
import {
  MUNKI_INSTALLER_TYPE_OPTIONS,
  MUNKI_INSTALLER_TYPE_VALUES,
} from "@/pages/munki/software/munki-software";

const PACKAGE_TYPE_FILTER_KEYS = [{ id: "type" }] as const;

function PackageSoftwareCell({ row }: CellContext<MunkiPackage, unknown>) {
  const { user } = useAuth();
  const content = (
    <>
      <SoftwareArtwork src={row.original.software.icon_url} />
      <span className="truncate">{row.original.software.name}</span>
    </>
  );
  return user?.role === "admin" ? (
    <Link
      to="/munki/packages/$id/edit"
      params={{ id: String(row.original.id) }}
      className="flex min-w-0 items-center gap-2 font-medium hover:underline"
    >
      {content}
    </Link>
  ) : (
    <span className="flex min-w-0 items-center gap-2 font-medium">{content}</span>
  );
}

const packageColumns: ColumnDef<MunkiPackage>[] = [
  selectColumn<MunkiPackage>(),
  {
    id: "software_name",
    accessorFn: (row) => row.software.name,
    header: "Software",
    cell: PackageSoftwareCell,
    enableHiding: false,
    meta: { label: "Software" },
  },
  {
    id: "version",
    accessorKey: "version",
    header: "Version",
    cell: ({ row }) => row.original.version,
    meta: { label: "Version" },
  },
  {
    id: "type",
    accessorKey: "installer_type",
    header: "Type",
    cell: ({ row }) => row.original.installer_type,
    enableColumnFilter: true,
    meta: { label: "Type", options: MUNKI_INSTALLER_TYPE_OPTIONS },
  },
  {
    id: "size",
    accessorFn: (row) => row.installer_file?.size_bytes ?? 0,
    header: "Size",
    cell: ({ row }) => {
      const bytes = row.original.installer_file?.size_bytes ?? 0;
      return bytes > 0 ? formatBytes(bytes) : "-";
    },
    meta: { label: "Size" },
  },
  {
    id: "updated_at",
    accessorKey: "updated_at",
    header: "Updated",
    cell: ({ row }) => formatRelative(row.original.updated_at),
    meta: { label: "Updated" },
  },
];

export function MunkiPackageListPage() {
  const tableSearch = useDataTableSearch(PACKAGE_TYPE_FILTER_KEYS);
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const packageTypes = (tableSearch.filters.type ?? []).filter((value) =>
    isOneOf(value, MUNKI_INSTALLER_TYPE_VALUES),
  );
  const query = useMunkiPackages({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    type: packageTypes,
  });
  const packages = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q || packageTypes.length > 0;
  const table = useDataTable({
    tableState: tableSearch,
    data: packages,
    columns: packageColumns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
    enableRowSelection: isAdmin,
  });
  return (
    <PageShell>
      <PageHeader
        title="Packages"
        actions={
          isAdmin ? (
            <Button size="sm" render={<Link to="/munki/packages/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />
      {query.error ? (
        <QueryError
          title="Failed to load packages"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={6} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          actionBar={
            isAdmin ? (
              <BulkDeleteActionBar
                table={table}
                useBulkDelete={useBulkDeleteMunkiPackages}
                noun="package"
                description="Packages still referenced by targeting or other packages cannot be deleted."
              />
            ) : undefined
          }
          empty={
            <DataTableEmpty
              icon={<PackageCheck />}
              filtered={hasFilters}
              filteredTitle="No matching packages"
              title="No packages"
              description="Create package versions for Munki software."
              filteredDescription="Try a different search."
            />
          }
        >
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput className="h-8 w-40 lg:w-56" />
              <DataTableFacetedFilter
                column={table.getColumn("type")}
                title="Type"
                options={MUNKI_INSTALLER_TYPE_OPTIONS}
                multiple
              />
            </div>
          </div>
        </DataTable>
      )}
    </PageShell>
  );
}
