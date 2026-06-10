import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { PackageSearch, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import {
  BulkDeleteDialog,
  DataTable,
  DataTableColumnHeader,
  DataTableEmptyState,
  DataTableSearch,
} from "@/components/data-table";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useBulkDeleteMunkiSoftware, useMunkiSoftware, type MunkiSoftware } from "@/hooks/use-munki-software";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { formatRelative } from "@/lib/utils";

export function MunkiSoftwareListPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [selectedSoftwareIds, setSelectedSoftwareIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const bulkDelete = useBulkDeleteMunkiSoftware();
  const query = useMunkiSoftware({
    q: typeof search.q === "string" ? search.q : undefined,
    ...tableQueryParams(state),
  });
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q;
  const selectedIDs = selectedSoftwareIds.map(Number);

  const deleteSelectedSoftware = () => {
    const count = selectedIDs.length;
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => {
        setSelectedSoftwareIds([]);
        setDeleteOpen(false);
        toast.success(`Deleted ${count} software`);
      },
    });
  };

  const columns: ColumnDef<MunkiSoftware>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Software" />,
      cell: ({ row }) => (
        <div className="flex min-w-0 items-center gap-2">
          <MunkiIcon iconUrl={row.original.icon_url} />
          <div className="min-w-0">
            <div className="truncate font-medium">{row.original.name}</div>
          </div>
        </div>
      ),
    },
    {
      id: "category",
      accessorKey: "category",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Category" />,
      cell: ({ row }) => row.original.category || "-",
    },
    {
      id: "developer",
      accessorKey: "developer",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Developer" />,
      cell: ({ row }) => row.original.developer || "-",
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
          enableRowSelection
          selectedRowIds={selectedSoftwareIds}
          onSelectedRowIdsChange={setSelectedSoftwareIds}
          bulkActions={
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)} disabled={bulkDelete.isPending}>
              <Trash2 data-icon="inline-start" />
              Delete
            </Button>
          }
          rowHref={(row) => ({
            to: "/munki/software/$softwareId",
            params: { softwareId: String(row.id) },
          })}
          toolbar={<DataTableSearch value={draft} onChange={setDraft} placeholder="Search" />}
          empty={
            <DataTableEmptyState
              icon={<PackageSearch />}
              title={hasFilters ? "No Matching Software" : "No Software"}
              description={hasFilters ? "Try a different search." : "Create software to manage Munki packages."}
            />
          }
        />
      )}
      <BulkDeleteDialog
        open={deleteOpen}
        onOpenChange={(open) => {
          if (!open) bulkDelete.reset();
          setDeleteOpen(open);
        }}
        count={selectedIDs.length}
        noun="software"
        description="Packages and targeting for the selected software will also be removed."
        pending={bulkDelete.isPending}
        onConfirm={deleteSelectedSoftware}
      />
    </PageShell>
  );
}
