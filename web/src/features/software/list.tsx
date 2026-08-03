import { getRouteApi } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { ChevronRight, Package } from "lucide-react";
import * as React from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import { useSoftware } from "@features/software/queries";
import { SoftwareIcon, softwareIconProps } from "@features/software/software-icon";
import {
  expandSoftwareSourceFilters,
  softwareSourceLabel,
  SOURCE_FILTER_OPTIONS,
  versionsSummaryLabel,
} from "@features/software/software-source-labels";
import type { SoftwareTitle } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";

const routeApi = getRouteApi("/_authenticated/software/");
const SOURCE_FILTER_KEYS = [{ id: "source", multiple: true }] as const;

export function SoftwareListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: SOURCE_FILTER_KEYS,
  });

  const sources = search.source ?? [];

  const query = useSoftware(
    {
      q: tableSearch.q,
      page: tableSearch.page,
      per_page: tableSearch.per_page,
      sort: tableSearch.sort,
      source: expandSoftwareSourceFilters(sources),
    },
    { refetchInterval: 30_000 },
  );

  const software = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const columns = React.useMemo<ColumnDef<SoftwareTitle>[]>(() => softwareColumns, []);

  const table = useDataTable({
    tableState: tableSearch,
    data: software,
    columns,
    pageCount,
    rowCount: totalCount,
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
        <DataTableSkeleton columnCount={5} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          empty={
            <DataTableEmpty
              icon={<Package />}
              filtered={tableSearch.isFiltered}
              title="No observed software"
              description="Inventory appears after hosts refresh."
              filteredDescription="No titles matched the current filters."
            />
          }
        >
          <DataTableSearchInput
            value={tableSearch.q ?? ""}
            onValueChange={tableSearch.onQueryChange}
          />
          <DataTableFacetedFilter
            column={table.getColumn("source")}
            title="Type"
            options={SOURCE_FILTER_OPTIONS}
          />
        </DataTable>
      )}
    </PageShell>
  );
}

const softwareColumns: ColumnDef<SoftwareTitle>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <div className="flex min-w-0 items-center gap-2">
        <SoftwareIcon {...softwareIconProps(row.original.source)} />
        <Link
          to="/software/titles/$id"
          params={{ id: String(row.original.id) }}
          className="min-w-0 truncate font-medium"
          title={row.original.name}
        >
          {row.original.name}
        </Link>
      </div>
    ),
    enableHiding: false,
    size: 320,
    minSize: 180,
    meta: { label: "Name" },
  },
  {
    id: "versions",
    accessorFn: (row) => row.versions.count,
    header: "Versions",
    cell: ({ row }) => versionsSummaryLabel(row.original.versions.items),
    meta: { label: "Versions" },
    size: 112,
  },
  {
    id: "source",
    accessorKey: "source",
    header: "Type",
    cell: ({ row }) => softwareSourceLabel(row.original.source, row.original.extension_for),
    meta: { label: "Type", options: SOURCE_FILTER_OPTIONS },
    enableColumnFilter: true,
    size: 160,
  },
  {
    id: "hosts_count",
    accessorKey: "hosts_count",
    header: "Hosts",
    cell: ({ row }) => row.original.hosts_count,
    meta: { label: "Hosts" },
    size: 96,
  },
  {
    id: "actions",
    header: () => null,
    cell: ({ row }) => (
      <Button
        variant="ghost"
        size="xs"
        render={<Link to="/hosts" search={{ software_title_id: row.original.id }} />}
        nativeButton={false}
      >
        See hosts
        <ChevronRight />
      </Button>
    ),
    enableSorting: false,
    enableHiding: false,
    size: 96,
    minSize: 96,
    maxSize: 96,
    enableResizing: false,
  },
];
