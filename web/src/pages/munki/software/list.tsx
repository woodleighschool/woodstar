import { Link } from "@tanstack/react-router";
import type { ColumnDef, Table as TanStackTable } from "@tanstack/react-table";
import { PackageSearch, Plus, Trash2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { BulkDeleteDialog } from "@/components/bulk-delete-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { useDataTable } from "@/hooks/use-data-table";
import { DEFAULT_PAGE_SIZE, useDataTableSearch } from "@/hooks/use-data-table-search";
import { useBulkDeleteMunkiSoftware, useMunkiSoftware, type MunkiSoftware } from "@/hooks/use-munki-software";
import { formatRelative } from "@/lib/utils";

export function MunkiSoftwareListPage() {
  const tableSearch = useDataTableSearch();

  const query = useMunkiSoftware({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });

  const software = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q;

  const columns = React.useMemo<ColumnDef<MunkiSoftware>[]>(
    () => [
      {
        id: "select",
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected() || (table.getIsSomePageRowsSelected() && "indeterminate")}
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
            aria-label="Select all"
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        ),
        enableSorting: false,
        enableHiding: false,
        size: 32,
      },
      {
        id: "name",
        accessorKey: "name",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Software" />,
        cell: ({ row }) => (
          <Link
            to="/munki/software/$softwareId"
            params={{ softwareId: String(row.original.id) }}
            className="flex min-w-0 items-center gap-2 font-medium hover:underline"
          >
            <MunkiIcon iconUrl={row.original.icon_url} />
            <span className="truncate">{row.original.name}</span>
          </Link>
        ),
        enableHiding: false,
        meta: { label: "Software" },
      },
      {
        id: "category",
        accessorKey: "category",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Category" />,
        cell: ({ row }) => row.original.category || "-",
        meta: { label: "Category" },
      },
      {
        id: "developer",
        accessorKey: "developer",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Developer" />,
        cell: ({ row }) => row.original.developer || "-",
        meta: { label: "Developer" },
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
    [],
  );

  const { table } = useDataTable({
    data: software,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  return (
    <PageShell>
      <PageHeader
        title="Software"
        actions={
          <Button asChild size="sm">
            <Link to="/munki/software/new">
              <Plus data-icon="inline-start" />
              Create
            </Link>
          </Button>
        }
      />

      {query.error ? (
        <QueryError title="Failed to load software" error={query.error} onRetry={() => void query.refetch()} />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={5} />
      ) : (
        <DataTable
          table={table}
          actionBar={<MunkiSoftwareActionBar table={table} />}
          empty={
            <Empty className="min-h-72 border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <PackageSearch />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matching software" : "No software"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters ? "Try a different search." : "Create software to manage Munki packages."}
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

function MunkiSoftwareActionBar({ table }: { table: TanStackTable<MunkiSoftware> }) {
  const rows = table.getFilteredSelectedRowModel().rows;
  const ids = React.useMemo(() => rows.map((row) => Number(row.original.id)), [rows]);
  const [open, setOpen] = React.useState(false);
  const bulkDelete = useBulkDeleteMunkiSoftware();

  const onConfirm = () => {
    const count = ids.length;
    bulkDelete.mutate(ids, {
      onSuccess: () => {
        toast.success(`Deleted ${count} software`);
        table.toggleAllRowsSelected(false);
        setOpen(false);
      },
    });
  };

  return (
    <div className="flex items-center gap-3 rounded-md border bg-background p-1 pl-3 shadow-sm">
      <span className="text-sm text-muted-foreground">{ids.length} selected</span>
      <Button variant="destructive" size="sm" onClick={() => setOpen(true)} disabled={bulkDelete.isPending}>
        <Trash2 />
        Delete
      </Button>
      <BulkDeleteDialog
        open={open}
        onOpenChange={(next) => {
          if (!next) bulkDelete.reset();
          setOpen(next);
        }}
        count={ids.length}
        noun="software"
        description="Packages and targeting for the selected software will also be removed."
        pending={bulkDelete.isPending}
        onConfirm={onConfirm}
      />
    </div>
  );
}
