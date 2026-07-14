import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Package } from "lucide-react";
import * as React from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { SoftwareIcon } from "@/components/software/software-icon";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { useSoftware } from "@/hooks/use-software";
import type { SoftwareTitle } from "@/lib/api";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import {
  expandSoftwareSourceFilters,
  softwareSourceLabel,
  SOURCE_FILTER_OPTIONS,
  versionsSummaryLabel,
} from "@/pages/software/software-source-labels";

const SOURCE_FILTER_KEYS = [{ id: "source" }] as const;

export function SoftwareListPage() {
  const tableSearch = useDataTableSearch(SOURCE_FILTER_KEYS);

  const sources = tableSearch.filters.source ?? [];

  const query = useSoftware({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    source: expandSoftwareSourceFilters(sources),
  });

  const software = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q || sources.length > 0;

  const columns = React.useMemo<ColumnDef<SoftwareTitle>[]>(() => softwareColumns, []);

  const table = useDataTable({
    tableState: tableSearch,
    data: software,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  return (
    <PageShell>
      <PageHeader
        title="Software"
        description="Search installed software and OS inventory observed across hosts."
      />

      {query.error ? (
        <QueryError
          title="Failed to load software"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={4} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          empty={
            <DataTableEmpty
              icon={<Package />}
              filtered={hasFilters}
              title="No observed software"
              description="Inventory appears after hosts refresh."
              filteredDescription="No titles matched the current filters."
            />
          }
        >
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput className="h-8 w-40 lg:w-56" />
              <DataTableFacetedFilter
                column={table.getColumn("source")}
                title="Type"
                options={SOURCE_FILTER_OPTIONS}
                multiple
              />
            </div>
          </div>
        </DataTable>
      )}
    </PageShell>
  );
}

const softwareColumns: ColumnDef<SoftwareTitle>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
    cell: ({ row }) => (
      <Link
        to="/software/titles/$softwareId"
        params={{ softwareId: String(row.original.id) }}
        className="inline-flex items-center gap-2 truncate font-medium hover:underline"
      >
        <SoftwareIcon source={row.original.source} />
        <span className="truncate">{row.original.name}</span>
      </Link>
    ),
    enableHiding: false,
    meta: { label: "Name" },
  },
  {
    id: "versions_count",
    accessorFn: (row) => row.versions_count,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Versions" />,
    cell: ({ row }) => versionsSummaryLabel(row.original.versions ?? []),
    meta: { label: "Versions" },
  },
  {
    id: "source",
    accessorKey: "source",
    header: ({ column }) => <DataTableColumnHeader column={column} label="Type" />,
    cell: ({ row }) => softwareSourceLabel(row.original.source, row.original.extension_for),
    meta: { label: "Type", options: SOURCE_FILTER_OPTIONS },
    enableColumnFilter: true,
  },
  {
    id: "hosts_count",
    accessorKey: "hosts_count",
    header: ({ column }) => <DataTableColumnHeader column={column} label="Hosts" />,
    cell: ({ row }) => row.original.hosts_count,
    meta: { label: "Hosts" },
  },
];
