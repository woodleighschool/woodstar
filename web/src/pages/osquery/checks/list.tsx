import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { CircleAlert, CircleCheck, Plus, ShieldCheck, Trash2 } from "lucide-react";
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
    {
      id: "query",
      accessorKey: "query",
      enableSorting: false,
      header: "Query",
      cell: ({ row }) => (
        <code className="text-muted-foreground block max-w-[34rem] truncate font-mono text-xs" title={row.original.query}>
          {row.original.query}
        </code>
      ),
    },
    {
      id: "passing_host_count",
      accessorKey: "passing_host_count",
      enableSorting: false,
      header: () => (
        <span className="flex items-center justify-end gap-1.5">
          <CircleCheck className="text-status-online size-4" />
          Pass
        </span>
      ),
      cell: ({ row }) => <HostCount value={row.original.passing_host_count} />,
      meta: { headClassName: "text-right" },
    },
    {
      id: "failing_host_count",
      accessorKey: "failing_host_count",
      enableSorting: false,
      header: () => (
        <span className="flex items-center justify-end gap-1.5">
          <CircleAlert className="text-muted-foreground size-4" />
          Fail
        </span>
      ),
      cell: ({ row }) => <HostCount value={row.original.failing_host_count} />,
      meta: { headClassName: "text-right" },
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title="Checks"
        description="Detect device health issues."
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
          <AlertTitle>Failed to Load Checks</AlertTitle>
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
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search" />
            </div>
          }
          empty={
            <DataTableEmptyState
              icon={<ShieldCheck />}
              title={hasFilters ? "No Matches" : "No Health Checks"}
              description={hasFilters ? "Try clearing the filters." : "Create a check from SQL."}
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

function HostCount({ value }: { value: number }) {
  return (
    <span className="block text-right tabular-nums">
      {value} {value === 1 ? "host" : "hosts"}
    </span>
  );
}
