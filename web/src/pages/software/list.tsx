import { useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Package, Search, X } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/ui/data-table";
import { DataTableColumnHeader } from "@/components/ui/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/ui/data-table-faceted-filter";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useSoftware, type SoftwareTitle } from "@/hooks/use-software";
import { useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { softwareSourceLabel, SOURCE_FILTER_OPTIONS } from "@/lib/software-source-labels";

export function SoftwarePage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");

  const sources = search.source ?? [];

  const query = useSoftware({
    q: search.q,
    source: sources,
    page: state.page,
    per_page: state.perPage,
    order_key: state.orderKey,
    order_direction: state.orderDirection,
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || sources.length > 0;

  const columns: ColumnDef<SoftwareTitle>[] = [
    {
      id: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => row.original.display_name || row.original.name,
    },
    {
      id: "versions_count",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Versions" />,
      cell: ({ row }) => {
        const versions = row.original.versions ?? [];
        const label =
          versions.length === 0
            ? "-"
            : versions.length === 1
              ? versions[0].version || "-"
              : `${versions.length} versions`;
        return <span className="text-muted-foreground tabular-nums">{label}</span>;
      },
    },
    {
      id: "source",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
      cell: ({ row }) => (
        <span className="text-muted-foreground" title={row.original.source}>
          {softwareSourceLabel(row.original.source, row.original.extension_for)}
        </span>
      ),
    },
    {
      id: "hosts_count",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Hosts" align="right" />,
      cell: ({ row }) => <div className="text-right tabular-nums">{row.original.hosts_count}</div>,
      meta: { headClassName: "text-right", cellClassName: "text-right" },
    },
  ];

  return (
    <div className="p-6">
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load software</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
          <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
            Retry
          </Button>
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
          rowHref={(row) => ({ to: "/software/titles/$softwareId", params: { softwareId: row.id } })}
          toolbar={
            <SoftwareToolbar
              draft={draft}
              onDraftChange={setDraft}
              sources={sources}
              onSourcesChange={(next) => setters.setFilter("source", next.length > 0 ? next.join(",") : undefined)}
              isFetching={query.isFetching}
              totalCount={totalCount}
            />
          }
          empty={
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Package />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matches" : "No software inventory yet"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters
                    ? "No titles matched the current filters."
                    : "Hosts will report installed apps and packages on their next detail refresh."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          }
        />
      )}
    </div>
  );
}

interface SoftwareToolbarProps {
  draft: string;
  onDraftChange: (next: string) => void;
  sources: string[];
  onSourcesChange: (next: string[]) => void;
  isFetching: boolean;
  totalCount: number;
}

function SoftwareToolbar({
  draft,
  onDraftChange,
  sources,
  onSourcesChange,
  isFetching,
  totalCount,
}: SoftwareToolbarProps) {
  return (
    <div className="flex items-center gap-2">
      <div className="relative max-w-md flex-1">
        <Search
          className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2"
          aria-hidden
        />
        <Input
          value={draft}
          onChange={(e) => onDraftChange(e.target.value)}
          placeholder="Search by name, display name, bundle id..."
          className="pr-8 pl-8"
          aria-label="Search software"
        />
        {draft ? (
          <button
            type="button"
            onClick={() => onDraftChange("")}
            className="text-muted-foreground hover:text-foreground absolute top-1/2 right-2 -translate-y-1/2 rounded p-0.5"
            aria-label="Clear search"
          >
            <X className="size-3.5" />
          </button>
        ) : null}
      </div>
      <DataTableFacetedFilter
        title="Type"
        options={SOURCE_FILTER_OPTIONS}
        selected={sources}
        onChange={onSourcesChange}
      />
      <div className="text-muted-foreground ml-auto text-xs tabular-nums">
        {isFetching ? "Loading..." : `${totalCount} ${totalCount === 1 ? "title" : "titles"}`}
      </div>
    </div>
  );
}
