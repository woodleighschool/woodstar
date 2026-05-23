import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { FileBarChart2, Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import { BulkDeleteDialog } from "@/components/data-table/bulk-delete-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { IntervalIndicator, PlatformBadge } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useBulkDeleteReports, useReports, type Report } from "@/hooks/use-reports";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { PLATFORM_LABELS, QUERYABLE_PLATFORMS } from "@/lib/targeting";

const PLATFORM_OPTIONS = QUERYABLE_PLATFORMS.map((platform) => ({ value: platform, label: PLATFORM_LABELS[platform] }));

export function ReportsPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [selectedReportIds, setSelectedReportIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const bulkDelete = useBulkDeleteReports();

  const reports = useReports({
    q: search.q,
    platform: search.platform,
    ...tableQueryParams(state),
  });

  const data = reports.data?.items ?? [];
  const totalCount = reports.data?.count ?? 0;
  const hasFilters = !!search.q || !!search.platform;
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
      id: "platform",
      header: "Targeted platforms",
      enableSorting: false,
      cell: ({ row }) => <PlatformBadge platforms={row.original.platforms} />,
    },
    {
      id: "schedule_interval",
      accessorKey: "schedule_interval",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Interval" />,
      cell: ({ row }) => <IntervalIndicator interval={row.original.schedule_interval} />,
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title="Reports"
        description="Gather data about your hosts."
        actions={
          <Button asChild size="sm">
            <Link to="/reports/new">
              <Plus data-icon="inline-start" />
              Add report
            </Link>
          </Button>
        }
      />
      {reports.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load reports</AlertTitle>
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
          rowHref={(row) => ({ to: "/reports/$reportId", params: { reportId: String(row.id) } })}
          toolbar={
            <div className="flex items-center gap-2">
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search by name" label="Search reports" />
              <DataTableFacetedFilter
                title="Platform"
                options={PLATFORM_OPTIONS}
                selected={search.platform ? [search.platform] : []}
                onChange={(next) => setters.setFilter("platform", next[0])}
                singleSelect
              />
            </div>
          }
          empty={
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <FileBarChart2 />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matches" : "No reports yet"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters
                    ? "Try clearing the filters."
                    : "Create a new report, or import Fleet's standard query library later."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
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
        error={bulkDelete.error?.message}
        pending={bulkDelete.isPending}
        onConfirm={deleteSelectedReports}
      />
    </PageShell>
  );
}
