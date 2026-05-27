import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Plus, ShieldCheck, Trash2 } from "lucide-react";
import { useState } from "react";

import { BulkDeleteDialog } from "@/components/data-table/bulk-delete-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmptyState } from "@/components/data-table/data-table-empty-state";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useBulkDeleteChecks, useChecks, type Check } from "@/hooks/use-checks";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";

export function ChecksPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [selectedCheckIds, setSelectedCheckIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const bulkDelete = useBulkDeleteChecks();

  const query = useChecks({
    q: search.q,
    ...tableQueryParams(state),
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q;
  const selectedIDs = selectedCheckIds.map(Number);

  const deleteSelectedChecks = () => {
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => {
        setSelectedCheckIds([]);
        setDeleteOpen(false);
      },
    });
  };

  const columns: ColumnDef<Check>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => (
        <div className="grid gap-1">
          <span className="font-medium">{row.original.name}</span>
          {row.original.description ? (
            <span className="text-muted-foreground line-clamp-2 text-xs">{row.original.description}</span>
          ) : null}
        </div>
      ),
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title="Checks"
        description="Detect device health issues and track which hosts need attention."
        actions={
          <Button asChild size="sm">
            <Link to="/osquery/checks/new">
              <Plus data-icon="inline-start" />
              Create
            </Link>
          </Button>
        }
      />
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load checks</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <DataTable
          columns={columns}
          data={data}
          totalCount={totalCount}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          enableRowSelection
          selectedRowIds={selectedCheckIds}
          onSelectedRowIdsChange={setSelectedCheckIds}
          bulkActions={
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)} disabled={bulkDelete.isPending}>
              <Trash2 data-icon="inline-start" />
              Delete
            </Button>
          }
          rowHref={(row) => ({ to: "/osquery/checks/$checkId", params: { checkId: String(row.id) } })}
          toolbar={
            <div className="flex items-center gap-2">
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search" label="Search checks" />
            </div>
          }
          empty={
            <DataTableEmptyState
              icon={<ShieldCheck />}
              title={hasFilters ? "No matches" : "No health checks"}
              description={
                hasFilters ? "Try clearing the filters." : "Create a check from SQL to track pass/fail host health."
              }
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
        noun="check"
        pending={bulkDelete.isPending}
        onConfirm={deleteSelectedChecks}
      />
    </PageShell>
  );
}
