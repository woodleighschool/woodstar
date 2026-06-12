import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { PackageCheck, Plus } from "lucide-react";
import * as React from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { useAuth } from "@/hooks/use-auth";
import { useDataTable } from "@/hooks/use-data-table";
import { DEFAULT_PAGE_SIZE, useDataTableSearch } from "@/hooks/use-data-table-search";
import { type MunkiPackage, useMunkiPackages } from "@/hooks/use-munki-packages";
import { formatRelative } from "@/lib/utils";
import { munkiInstallerTypeLabel } from "../software/munki-software";

export function MunkiPackageListPage() {
  const tableSearch = useDataTableSearch();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";

  const query = useMunkiPackages({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });

  const packages = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q;

  const columns = React.useMemo<ColumnDef<MunkiPackage>[]>(
    () => [
      {
        id: "package",
        accessorKey: "software_name",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Package" />,
        cell: ({ row }) =>
          isAdmin ? (
            <Link
              to="/munki/packages/$packageId/edit"
              params={{ packageId: String(row.original.id) }}
              className="flex min-w-0 items-center gap-2 font-medium hover:underline"
            >
              <MunkiIcon iconUrl={row.original.icon_url} />
              <span className="truncate">{row.original.software_name}</span>
            </Link>
          ) : (
            <span className="flex min-w-0 items-center gap-2 font-medium">
              <MunkiIcon iconUrl={row.original.icon_url} />
              <span className="truncate">{row.original.software_name}</span>
            </span>
          ),
        enableHiding: false,
        meta: { label: "Package" },
      },
      {
        id: "version",
        accessorKey: "version",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Version" />,
        cell: ({ row }) => row.original.version,
        meta: { label: "Version" },
      },
      {
        id: "type",
        accessorKey: "installer_type",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Type" />,
        cell: ({ row }) => munkiInstallerTypeLabel(row.original.installer_type),
        meta: { label: "Type" },
      },
      {
        id: "updated_at",
        accessorKey: "updated_at",
        header: () => "Updated",
        enableSorting: false,
        cell: ({ row }) => formatRelative(row.original.updated_at),
        meta: { label: "Updated" },
      },
    ],
    [isAdmin],
  );

  const { table } = useDataTable({
    data: packages,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  return (
    <PageShell>
      <PageHeader
        title="Packages"
        actions={
          isAdmin ? (
            <Button asChild size="sm">
              <Link to="/munki/packages/new">
                <Plus data-icon="inline-start" />
                Create
              </Link>
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
        <DataTableSkeleton columnCount={4} />
      ) : (
        <DataTable
          table={table}
          empty={
            <Empty className="min-h-72 border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <PackageCheck />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matching packages" : "No packages"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters
                    ? "Try a different search."
                    : "Create package versions for Munki software."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
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
