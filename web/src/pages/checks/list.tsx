import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Plus, ShieldCheck, Trash2 } from "lucide-react";
import { useState } from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageLead, PlatformBadge } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { useBulkDeleteChecks, useChecks, type Check } from "@/hooks/use-checks";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { PLATFORM_LABELS, QUERYABLE_PLATFORMS } from "@/lib/targeting";

const PLATFORM_OPTIONS = QUERYABLE_PLATFORMS.map((platform) => ({ value: platform, label: PLATFORM_LABELS[platform] }));

export function ChecksPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [selectedCheckIds, setSelectedCheckIds] = useState<string[]>([]);
  const bulkDelete = useBulkDeleteChecks();

  const query = useChecks({
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
  const selectedIDs = selectedCheckIds.map(Number);

  const deleteSelectedChecks = () => {
    const count = selectedIDs.length;
    if (count === 0) return;
    if (!window.confirm(`Delete ${count} selected ${count === 1 ? "check" : "checks"}?`)) return;
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => setSelectedCheckIds([]),
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
      id: "platform",
      header: "Platforms",
      enableSorting: false,
      cell: ({ row }) => <PlatformBadge platform={row.original.platform} />,
    },
  ];

  return (
    <div className="flex flex-col gap-5 p-6">
      <PageLead
        title="Checks"
        description="Detect device health issues and track which hosts need attention."
        actions={
          <Button asChild size="sm">
            <Link to="/checks/new">
              <Plus data-icon="inline-start" />
              Add check
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
          page={state.page}
          perPage={state.perPage}
          sort={{ orderKey: state.orderKey, orderDirection: state.orderDirection }}
          onPageChange={setters.setPage}
          onPerPageChange={setters.setPerPage}
          onSortChange={(s) => setters.setSort(s.orderKey, s.orderDirection)}
          isLoading={query.isLoading}
          enableRowSelection
          selectedRowIds={selectedCheckIds}
          onSelectedRowIdsChange={setSelectedCheckIds}
          bulkActions={
            <Button variant="destructive" size="sm" onClick={deleteSelectedChecks} disabled={bulkDelete.isPending}>
              <Trash2 data-icon="inline-start" />
              Delete
            </Button>
          }
          rowHref={(row) => ({ to: "/checks/$checkId", params: { checkId: String(row.id) } })}
          toolbar={
            <div className="flex items-center gap-2">
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search by name" label="Search checks" />
              <DataTableFacetedFilter
                title="Platform"
                options={PLATFORM_OPTIONS}
                selected={search.platform ? [search.platform] : []}
                onChange={(next) => setters.setFilter("platform", next[0])}
                singleSelect
              />
              <div className="text-muted-foreground ml-auto text-xs tabular-nums">
                {query.isFetching ? "Loading..." : `${totalCount} ${totalCount === 1 ? "check" : "checks"}`}
              </div>
            </div>
          }
          empty={
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <ShieldCheck />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matches" : "No checks yet"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters
                    ? "Try clearing the filters."
                    : "Add checks to detect device health issues and trigger follow-up workflows later."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          }
        />
      )}
    </div>
  );
}
