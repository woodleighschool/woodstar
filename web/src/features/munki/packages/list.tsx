import { getRouteApi } from "@tanstack/react-router";
import { filesize } from "filesize";
import { MoreHorizontal, PackageCheck, Plus } from "lucide-react";
import * as React from "react";

import { BulkDeleteActionBar } from "@components/bulk-delete-action-bar";
import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
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
import { useCan } from "@features/authz/access";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { MunkiPackage } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { formatRelative } from "@lib/utils";

import { MUNKI_INSTALLER_TYPE_OPTIONS } from "../software/metadata";
import { MunkiPackageDeleteDialog } from "./delete-dialog";
import { useBulkDeleteMunkiPackages, useMunkiPackages } from "./queries";

const routeApi = getRouteApi("/_authenticated/munki/packages/");
const PACKAGE_TYPE_FILTER_KEYS = [{ id: "type", multiple: true }] as const;

function PackageSoftwareCell({ row }: DataTableCellContext<MunkiPackage>) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <SoftwareArtwork src={row.original.software.icon_url} />
      <TextLink
        to="/munki/packages/$id"
        params={{ id: String(row.original.id) }}
        className="min-w-0 truncate font-medium"
        title={row.original.software.name}
      >
        {row.original.software.name}
      </TextLink>
    </div>
  );
}

function packageColumns(
  canEdit: boolean,
  onDelete: (pkg: MunkiPackage) => void,
): DataTableColumnDef<MunkiPackage>[] {
  const columns: DataTableColumnDef<MunkiPackage>[] = [
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
        return bytes > 0 ? filesize(bytes) : "-";
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
  if (!canEdit) return columns;
  return [
    selectColumn<MunkiPackage>(),
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
        <PackageRowActions pkg={row.original} onDelete={() => onDelete(row.original)} />
      ),
    },
  ];
}

export function MunkiPackageListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: PACKAGE_TYPE_FILTER_KEYS,
  });
  const canEdit = useCan({ resource: "munki.packages", access: "edit" });
  const bulkDelete = useBulkDeleteMunkiPackages();
  const [deleting, setDeleting] = React.useState<MunkiPackage | null>(null);
  const packageTypes = search.type ?? [];
  const query = useMunkiPackages({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    type: packageTypes,
  });
  const packages = React.useMemo(() => query.data?.items ?? [], [query.data?.items]);
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const columns = React.useMemo(() => packageColumns(canEdit, setDeleting), [canEdit]);
  const table = useDataTable({
    tableState: tableSearch,
    data: packages,
    columns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
    enableRowSelection: canEdit,
  });
  return (
    <PageShell>
      <PageHeader
        title="Packages"
        actions={
          canEdit ? (
            <Button size="sm" render={<Link to="/munki/packages/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />
      {query.error ? (
        <QueryError
          title="Failed to Load Packages"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={canEdit ? 7 : 5} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          pending={query.isPlaceholderData}
          actionBar={
            canEdit ? (
              <BulkDeleteActionBar
                table={table}
                bulkDelete={bulkDelete}
                noun="package"
                description="Packages still referenced by targeting or other packages cannot be deleted."
              />
            ) : undefined
          }
          empty={
            <DataTableEmpty
              icon={<PackageCheck />}
              filtered={tableSearch.isFiltered}
              filteredTitle="No Matching Packages"
              title="No Packages"
              description="Create package versions for Munki software."
              filteredDescription="Try a different search."
            />
          }
        >
          <DataTableSearchInput
            loading={query.isPlaceholderData}
            value={tableSearch.q ?? ""}
            onValueChange={tableSearch.onQueryChange}
          />
          <DataTableFacetedFilter
            column={table.getColumn("type")}
            title="Type"
            options={MUNKI_INSTALLER_TYPE_OPTIONS}
          />
        </DataTable>
      )}

      {canEdit ? (
        <MunkiPackageDeleteDialog
          pkg={deleting}
          open={deleting !== null}
          onOpenChange={(open) => {
            if (!open) setDeleting(null);
          }}
        />
      ) : null}
    </PageShell>
  );
}

function PackageRowActions({ pkg, onDelete }: { pkg: MunkiPackage; onDelete: () => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button type="button" size="icon" variant="ghost" />}>
        <MoreHorizontal />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem
            render={<Link to="/munki/packages/$id/edit" params={{ id: String(pkg.id) }} />}
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
