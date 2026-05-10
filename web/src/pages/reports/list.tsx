import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { FileBarChart2, Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { IntervalIndicator, PageLead, PlatformBadge, TargetSummary } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useBulkDeleteQueries, useQueries, type SavedQuery } from "@/hooks/use-queries";
import { useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { PLATFORM_LABELS, QUERYABLE_PLATFORMS } from "@/lib/targeting";
import { formatRelative } from "@/lib/utils";

const PLATFORM_OPTIONS = QUERYABLE_PLATFORMS.map((platform) => ({ value: platform, label: PLATFORM_LABELS[platform] }));

export function ReportsPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [selectedReportIds, setSelectedReportIds] = useState<string[]>([]);
  const bulkDelete = useBulkDeleteQueries();

  const query = useQueries({
    q: search.q,
    platform: search.platform,
    page: state.page,
    per_page: state.perPage,
    order_key: state.orderKey,
    order_direction: state.orderDirection,
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!search.platform;
  const selectedIDs = selectedReportIds.map(Number);

  const deleteSelectedReports = () => {
    const count = selectedIDs.length;
    if (count === 0) return;
    if (!window.confirm(`Delete ${count} selected ${count === 1 ? "report" : "reports"}?`)) return;
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => setSelectedReportIds([]),
    });
  };

  const columns: ColumnDef<SavedQuery>[] = [
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
      cell: ({ row }) => <PlatformBadge platform={row.original.platform} />,
    },
    {
      id: "targets",
      header: () => "Targets",
      enableSorting: false,
      cell: ({ row }) => <TargetSummary scope={row.original.label_scope} platform={row.original.platform} />,
    },
    {
      id: "schedule_interval",
      accessorKey: "schedule_interval",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Interval" />,
      cell: ({ row }) => <IntervalIndicator interval={row.original.schedule_interval} />,
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Last modified" />,
      cell: ({ row }) => <span className="text-muted-foreground">{formatRelative(row.original.updated_at)}</span>,
    },
  ];

  return (
    <div className="flex flex-col gap-5 p-6">
      <PageLead
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
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load reports</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <DataTable
          columns={columns}
          data={data}
          totalCount={totalCount}
          page={state.page}
          perPage={state.perPage}
          sort={{ orderKey: state.orderKey, orderDirection: state.orderDirection }}
          onPageChange={setters.setPage}
          onPerPageChange={setters.setPerPage}
          onSortChange={(s) => setters.setSort(s.orderKey, s.orderDirection)}
          isLoading={query.isLoading}
          enableRowSelection
          selectedRowIds={selectedReportIds}
          onSelectedRowIdsChange={setSelectedReportIds}
          bulkActions={
            <Button variant="destructive" size="sm" onClick={deleteSelectedReports} disabled={bulkDelete.isPending}>
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
              <div className="text-muted-foreground ml-auto text-xs tabular-nums">
                {query.isFetching ? "Loading..." : `${totalCount} ${totalCount === 1 ? "report" : "reports"}`}
              </div>
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
    </div>
  );
}
