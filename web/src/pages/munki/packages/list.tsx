import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { PackageCheck, Plus } from "lucide-react";

import { DataTable, DataTableColumnHeader, DataTableEmptyState, DataTableSearch } from "@/components/data-table";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useMunkiPackages, type MunkiPackage } from "@/hooks/munki/packages";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { munkiInstallerTypeLabel } from "@/lib/munki-software";
import { formatRelative } from "@/lib/utils";

export function MunkiPackagesPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const query = useMunkiPackages({
    q: typeof search.q === "string" ? search.q : undefined,
    ...tableQueryParams(state),
  });
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q;

  const columns: ColumnDef<MunkiPackage>[] = [
    {
      id: "package",
      accessorKey: "software_name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Package" />,
      cell: ({ row }) => (
        <div className="flex min-w-0 items-center gap-2">
          <MunkiIcon iconUrl={row.original.icon_url} />
          <div className="truncate font-medium">{row.original.software_name}</div>
        </div>
      ),
    },
    {
      id: "version",
      accessorKey: "version",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Version" />,
      cell: ({ row }) => row.original.version,
    },
    {
      id: "type",
      accessorKey: "installer_type",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
      cell: ({ row }) => munkiInstallerTypeLabel(row.original.installer_type),
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: "Updated",
      enableSorting: false,
      cell: ({ row }) => formatRelative(row.original.updated_at),
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title="Packages"
        actions={
          <Button asChild size="sm">
            <Link to="/munki/packages/new">
              <Plus data-icon="inline-start" />
              Create
            </Link>
          </Button>
        }
      />
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Packages</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          totalCount={totalCount}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          rowHref={(row) => ({
            to: "/munki/packages/$packageId/edit",
            params: { packageId: String(row.id) },
          })}
          toolbar={<DataTableSearch value={draft} onChange={setDraft} placeholder="Search" />}
          empty={
            <DataTableEmptyState
              icon={<PackageCheck />}
              title={hasFilters ? "No Matching Packages" : "No Packages"}
              description={hasFilters ? "Try a different search." : "Create package versions for Munki software."}
            />
          }
        />
      )}
    </PageShell>
  );
}
