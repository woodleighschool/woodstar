import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { FileBarChart2, Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import {
  BulkDeleteDialog,
  DataTable,
  DataTableColumnHeader,
  DataTableEmptyState,
  DataTableSearch,
} from "@/components/data-table";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useBulkDeleteReports, useReports, type Report } from "@/hooks/use-reports";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { formatInterval } from "@/lib/utils";

export function ReportsPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [selectedReportIds, setSelectedReportIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const bulkDelete = useBulkDeleteReports();

  const reports = useReports({
    q: search.q,
    ...tableQueryParams(state),
  });

  const data = reports.data?.items ?? [];
  const totalCount = reports.data?.count ?? 0;
  const hasFilters = !!search.q;
  const selectedIDs = selectedReportIds.map(Number);

  const deleteSelectedReports = () => {
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => {
        setSelectedReportIds([]);
        setDeleteOpen(false);
      },
    });
  };

  const columns: ColumnDef<Report>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => row.original.name,
    },
    {
      id: "schedule_interval",
      accessorKey: "schedule_interval",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Interval" />,
      cell: ({ row }) =>
        row.original.schedule_interval ? `Every ${formatInterval(row.original.schedule_interval)}` : "Off",
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title="Reports"
        actions={
          <Button asChild size="sm">
            <Link to="/osquery/reports/new">
              <Plus data-icon="inline-start" />
              Create
            </Link>
          </Button>
        }
      />
      {reports.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Reports</AlertTitle>
          <AlertDescription>{reports.error.message}</AlertDescription>
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
          isLoading={reports.isLoading}
          enableRowSelection
          selectedRowIds={selectedReportIds}
          onSelectedRowIdsChange={setSelectedReportIds}
          bulkActions={
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)} disabled={bulkDelete.isPending}>
              <Trash2 data-icon="inline-start" />
              Delete
            </Button>
          }
          rowHref={(row) => ({ to: "/osquery/reports/$reportId", params: { reportId: String(row.id) } })}
          toolbar={
            <div className="flex items-center gap-2">
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search" />
            </div>
          }
          empty={
            <DataTableEmptyState
              icon={<FileBarChart2 />}
              title={hasFilters ? "No Matches" : "No Saved Queries"}
              description={hasFilters ? "Try clearing the filters." : "Create a report from SQL."}
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
        noun="report"
        pending={bulkDelete.isPending}
        onConfirm={deleteSelectedReports}
      />
    </PageShell>
  );
}
