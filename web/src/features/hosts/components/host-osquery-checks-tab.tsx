import { getRouteApi } from "@tanstack/react-router";
import { useMemo } from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableColumnDef, DataTableInstance } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { useHostOsqueryChecks } from "@features/hosts/queries";
import { CHECK_RESULT_STATUSES, CHECK_RESULT_STATUS_OPTIONS } from "@features/osquery/checks/model";
import type { OsqueryCheckHostStatus } from "@lib/api";
import { formatRelative } from "@lib/utils";

const checkColumns: DataTableColumnDef<OsqueryCheckHostStatus>[] = [
  {
    accessorKey: "check_name",
    header: () => "Check",
    cell: ({ row }) => (
      <Link to="/osquery/checks/$id" params={{ id: String(row.original.check_id) }}>
        {row.original.check_name}
      </Link>
    ),
  },
  {
    accessorKey: "status",
    header: () => "Status",
    enableColumnFilter: true,
    cell: ({ row }) => (
      <EnumStatusIndicator value={row.original.status} metadata={CHECK_RESULT_STATUSES} />
    ),
  },
  {
    accessorKey: "updated_at",
    header: () => "Last Evaluated",
    cell: ({ row }) => (row.original.updated_at ? formatRelative(row.original.updated_at) : "-"),
  },
];

const STATUS_FILTER_KEYS = [{ id: "status", multiple: true }] as const;
const routeApi = getRouteApi("/_authenticated/hosts/$id/checks");

function HostChecksToolbar({ table }: { table: DataTableInstance<OsqueryCheckHostStatus> }) {
  return (
    <DataTableFacetedFilter
      column={table.getColumn("status")}
      title="Status"
      options={CHECK_RESULT_STATUS_OPTIONS}
    />
  );
}

export function HostOsqueryChecksTab({ hostId }: { hostId: number | null }) {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: STATUS_FILTER_KEYS,
  });
  const status = search.status;
  const query = useHostOsqueryChecks(hostId, {
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    status,
  });
  const rows = useMemo(() => query.data?.items ?? [], [query.data?.items]);
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: rows,
    columns: checkColumns,
    pageCount,
    rowCount: totalCount,
    getRowId: (row) => String(row.check_id),
  });

  if (hostId === null) {
    return (
      <QueryError title="Failed to load checks" error={{ message: "Host route is invalid." }} />
    );
  }

  if (query.error) {
    return (
      <QueryError
        title="Failed to load checks"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (query.isLoading) {
    return <DataTableSkeleton columnCount={3} filterCount={1} />;
  }

  return (
    <DataTable
      table={table}
      pending={query.isFetching}
      heading="Checks"
      empty={
        <PanelEmptyState>
          {tableSearch.isFiltered ? "No matching checks" : "No checks yet"}
        </PanelEmptyState>
      }
    >
      <DataTableSearchInput
        loading={query.isFetching}
        value={tableSearch.q ?? ""}
        onValueChange={tableSearch.onQueryChange}
        placeholder="Search checks"
      />
      <HostChecksToolbar table={table} />
    </DataTable>
  );
}
