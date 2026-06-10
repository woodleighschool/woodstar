import { useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Package } from "lucide-react";
import type { ReactNode } from "react";

import {
  DataTable,
  DataTableColumnHeader,
  DataTableEmptyState,
  DataTableFacetedFilter,
  DataTableSearch,
} from "@/components/data-table";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { SoftwareIcon } from "@/components/software/software-icon";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useSoftware, type SoftwareTitle } from "@/hooks/use-software";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import {
  expandSoftwareSourceFilters,
  softwareSourceLabel,
  SOURCE_FILTER_OPTIONS,
  versionsSummaryLabel,
} from "@/lib/software-source-labels";

export function SoftwareListPage() {
  const search = useSearch({ from: "/_authenticated/software/" });
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
      cell: ({ row }) => {
        const name = row.original.display_name || row.original.name;
        return (
          <span className="inline-flex items-center gap-2 truncate">
            <SoftwareIcon source={row.original.source} />
            <span className="truncate">{name}</span>
          </span>
        );
      },
    },
    {
      id: "versions_count",
      accessorFn: (row) => row.versions_count,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Versions" />,
      cell: ({ row }) => versionsSummaryLabel(row.original.versions ?? []),
    },
    {
      id: "source",
      accessorKey: "source",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
      cell: ({ row }) => softwareSourceLabel(row.original.source, row.original.extension_for),
    },
    {
      id: "hosts_count",
      accessorKey: "hosts_count",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Hosts" align="right" />,
      cell: ({ row }) => row.original.hosts_count,
      meta: { headClassName: "text-right", cellClassName: "text-right" },
    },
  ];

  return (
    <PageShell>
      <PageHeader title="Software" description="Manage software and search for installed software and OS inventory." />

      {query.error ? (
        <QueryError title="Failed to load software" error={query.error} onRetry={() => void query.refetch()} />
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
          showExport
          exportFilename="software.csv"
          rowHref={(row) => ({ to: "/software/titles/$softwareId", params: { softwareId: String(row.id) } })}
          toolbar={(_table, exportButton) => (
            <SoftwareToolbar
              draft={draft}
              onDraftChange={setDraft}
              sources={sources}
              onSourcesChange={(next) => setters.setFilter("source", next.length > 0 ? next.join(",") : undefined)}
              actions={exportButton}
            />
          )}
          empty={
            <DataTableEmptyState
              icon={<Package />}
              title={hasFilters ? "No Matches" : "No Observed Software"}
              description={
                hasFilters ? "No titles matched the current filters." : "Inventory appears after hosts refresh."
              }
            />
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
  actions?: ReactNode;
}

function SoftwareToolbar({ draft, onDraftChange, sources, onSourcesChange, actions }: SoftwareToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <DataTableSearch value={draft} onChange={onDraftChange} placeholder="Search" />
      <DataTableFacetedFilter
        title="Type"
        options={SOURCE_FILTER_OPTIONS}
        selected={sources}
        onChange={onSourcesChange}
      />
      {actions ? <div className="ml-auto">{actions}</div> : null}
    </div>
  );
}
