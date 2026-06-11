import { Link } from "@tanstack/react-router";
import type { ColumnDef, Table as TanStackTable } from "@tanstack/react-table";
import { FileBarChart2, Plus, Trash2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { BulkDeleteDialog } from "@/components/bulk-delete-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { useDataTable } from "@/hooks/use-data-table";
import { DEFAULT_PAGE_SIZE, useDataTableSearch } from "@/hooks/use-data-table-search";
import { useBulkDeleteReports, useReports, type Report } from "@/hooks/use-reports";
import { formatInterval } from "@/lib/utils";

export function ReportListPage() {
  const tableSearch = useDataTableSearch();

  const query = useReports({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });

  const reports = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q;

  const columns = React.useMemo<ColumnDef<Report>[]>(() => reportColumns, []);

  const { table } = useDataTable({
    data: reports,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

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
      {query.error ? (
        <QueryError title="Failed to load reports" error={query.error} onRetry={() => void query.refetch()} />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={3} />
      ) : (
        <DataTable
          table={table}
          actionBar={<ReportsActionBar table={table} />}
          empty={
            <Empty className="min-h-72 border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <FileBarChart2 />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matches" : "No saved queries"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters ? "No reports matched the current search." : "Create a report from SQL."}
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

const reportColumns: ColumnDef<Report>[] = [
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
    header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
    cell: ({ row }) => (
      <Link
        to="/osquery/reports/$reportId"
        params={{ reportId: String(row.original.id) }}
        className="font-medium hover:underline"
      >
        {row.original.name}
      </Link>
    ),
    enableHiding: false,
    meta: { label: "Name" },
  },
  {
    id: "schedule_interval",
    accessorKey: "schedule_interval",
    header: ({ column }) => <DataTableColumnHeader column={column} label="Interval" />,
    cell: ({ row }) =>
      row.original.schedule_interval ? `Every ${formatInterval(row.original.schedule_interval)}` : "Off",
    meta: { label: "Interval" },
  },
];

function ReportsActionBar({ table }: { table: TanStackTable<Report> }) {
  const rows = table.getFilteredSelectedRowModel().rows;
  const ids = React.useMemo(() => rows.map((row) => Number(row.original.id)), [rows]);
  const [open, setOpen] = React.useState(false);
  const bulkDelete = useBulkDeleteReports();

  const onConfirm = () => {
    const count = ids.length;
    bulkDelete.mutate(ids, {
      onSuccess: () => {
        toast.success(`Deleted ${count} ${count === 1 ? "report" : "reports"}`);
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
        noun="report"
        pending={bulkDelete.isPending}
        onConfirm={onConfirm}
      />
    </div>
  );
}
