import { getRouteApi } from "@tanstack/react-router";
import type { ColumnDef, Table } from "@tanstack/react-table";

import { DataTableClient } from "@components/data-table/data-table-client";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { useHostOsqueryChecks } from "@features/hosts/queries";
import {
  CHECK_RESULT_STATUSES,
  CHECK_RESULT_STATUS_OPTIONS,
  parseCheckResultStatus,
} from "@features/osquery/checks/model";
import type { OsqueryCheckHostStatus } from "@lib/api";
import { formatRelative } from "@lib/utils";

const checkColumns: ColumnDef<OsqueryCheckHostStatus>[] = [
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
    filterFn: (row, id, value: string[]) => value.includes(row.getValue(id)),
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

const STATUS_FILTER_KEYS = [{ id: "status" }] as const;
const routeApi = getRouteApi("/_authenticated/hosts/$id/checks");

function HostChecksToolbar({ table }: { table: Table<OsqueryCheckHostStatus> }) {
  return (
    <DataTableFacetedFilter
      column={table.getColumn("status")}
      title="Status"
      options={CHECK_RESULT_STATUS_OPTIONS}
    />
  );
}

function renderHostChecksToolbar(table: Table<OsqueryCheckHostStatus>) {
  return <HostChecksToolbar table={table} />;
}

export function HostOsqueryChecksTab({ hostId }: { hostId: number | null }) {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: STATUS_FILTER_KEYS,
  });
  const status = parseCheckResultStatus(tableSearch.filters.status?.[0]);
  const query = useHostOsqueryChecks(hostId, { status });
  const rows = query.data ?? [];

  if (query.error) {
    return (
      <QueryError
        title="Failed to load checks"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (query.isLoading) return null;

  return (
    <DataTableClient
      title="Checks"
      columns={checkColumns}
      data={rows}
      searchPlaceholder="Search checks"
      tableState={tableSearch}
      toolbar={renderHostChecksToolbar}
      empty={
        <PanelEmptyState>
          {status ? "No checks match this status" : "No checks yet"}
        </PanelEmptyState>
      }
    />
  );
}
