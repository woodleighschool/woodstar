import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { PackageSearch, Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import {
  BulkDeleteDialog,
  DataTable,
  DataTableColumnHeader,
  DataTableEmptyState,
  DataTableSearch,
} from "@/components/data-table";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  useBulkDeleteMunkiSoftwareTitles,
  useMunkiSoftwareTitles,
  type MunkiSoftwareTitle,
} from "@/hooks/munki/software-titles";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { formatRelative } from "@/lib/utils";

export function MunkiSoftwareTitlesPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [selectedSoftwareTitleIds, setSelectedSoftwareTitleIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const bulkDelete = useBulkDeleteMunkiSoftwareTitles();
  const query = useMunkiSoftwareTitles({
    q: typeof search.q === "string" ? search.q : undefined,
    ...tableQueryParams(state),
  });
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q;
  const selectedIDs = selectedSoftwareTitleIds.map(Number);

  const deleteSelectedSoftwareTitles = () => {
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => {
        setSelectedSoftwareTitleIds([]);
        setDeleteOpen(false);
      },
    });
  };

  const columns: ColumnDef<MunkiSoftwareTitle>[] = [
    {
      id: "name",
      accessorKey: "display_name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Software" />,
      cell: ({ row }) => (
        <div className="flex min-w-0 items-center gap-2">
          <MunkiIcon iconUrl={row.original.icon_url} />
          <div className="min-w-0">
            <div className="truncate font-medium">{row.original.display_name || row.original.name}</div>
            <div className="text-muted-foreground truncate text-xs">{row.original.name}</div>
          </div>
        </div>
      ),
    },
    {
      id: "category",
      accessorKey: "category",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Category" />,
      cell: ({ row }) => row.original.category || "Uncategorised",
    },
    {
      id: "developer",
      accessorKey: "developer",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Developer" />,
      cell: ({ row }) => row.original.developer || "Unknown",
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
            <Link to="/munki/software-titles/new">
              <Plus data-icon="inline-start" />
              Create
            </Link>
          </Button>
        }
      />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Software</AlertTitle>
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
          enableRowSelection
          selectedRowIds={selectedSoftwareTitleIds}
          onSelectedRowIdsChange={setSelectedSoftwareTitleIds}
          bulkActions={
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)} disabled={bulkDelete.isPending}>
              <Trash2 data-icon="inline-start" />
              Delete
            </Button>
          }
          rowHref={(row) => ({
            to: "/munki/software-titles/$softwareId",
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
        noun="software title"
        description="Packages and assignment targeting for the selected titles will also be removed."
        pending={bulkDelete.isPending}
        onConfirm={deleteSelectedSoftwareTitles}
      />
    </PageShell>
  );
}
