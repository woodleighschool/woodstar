import { useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Package } from "lucide-react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageShell } from "@/components/layout/page-layout";
import { SoftwareIcon } from "@/components/software/software-icon";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useSoftware, type SoftwareTitle } from "@/hooks/use-software";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { expandSoftwareSourceFilters, softwareSourceLabel, SOURCE_FILTER_OPTIONS } from "@/lib/software-source-labels";

export function SoftwarePage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");

  const sources = search.source ?? [];

  const query = useSoftware({
    q: search.q,
    source: expandSoftwareSourceFilters(sources),
    ...tableQueryParams(state),
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || sources.length > 0;

  const columns: ColumnDef<SoftwareTitle>[] = [
    {
      id: "name",
      accessorFn: (row) => row.display_name || row.name,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => (
        <span className="inline-flex items-center gap-2 truncate">
          <SoftwareIcon source={row.original.source} />
          <span className="truncate">{row.original.display_name || row.original.name}</span>
        </span>
      ),
    },
    {
      id: "versions_count",
      accessorFn: (row) => row.versions?.length ?? 0,
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
      accessorKey: "source",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
      cell: ({ row }) => (
        <span className="text-muted-foreground" title={row.original.source}>
          {softwareSourceLabel(row.original.source, row.original.extension_for)}
        </span>
      ),
    },
    {
      id: "hosts_count",
      accessorKey: "hosts_count",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Hosts" align="right" />,
      cell: ({ row }) => <div className="text-right tabular-nums">{row.original.hosts_count}</div>,
      meta: { headClassName: "text-right", cellClassName: "text-right" },
    },
  ];

  return (
    <PageShell>
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
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          rowHref={(row) => ({ to: "/software/titles/$softwareId", params: { softwareId: String(row.id) } })}
          toolbar={
            <SoftwareToolbar
              draft={draft}
              onDraftChange={setDraft}
              sources={sources}
              onSourcesChange={(next) => setters.setFilter("source", next.length > 0 ? next.join(",") : undefined)}
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
    </PageShell>
  );
}

interface SoftwareToolbarProps {
  draft: string;
  onDraftChange: (next: string) => void;
  sources: string[];
  onSourcesChange: (next: string[]) => void;
}

function SoftwareToolbar({ draft, onDraftChange, sources, onSourcesChange }: SoftwareToolbarProps) {
  return (
    <div className="flex items-center gap-2">
      <DataTableSearch
        value={draft}
        onChange={onDraftChange}
        placeholder="Search by name, display name, bundle id..."
        label="Search software"
      />
      <DataTableFacetedFilter
        title="Type"
        options={SOURCE_FILTER_OPTIONS}
        selected={sources}
        onChange={onSourcesChange}
      />
    </div>
  );
}
