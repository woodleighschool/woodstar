import { getRouteApi } from "@tanstack/react-router";
import { useMemo } from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableColumnDef, DataTableInstance } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { TextLink } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { useHostOsqueryPolicies } from "@features/hosts/queries";
import { POLICY_RESULT_STATUS_OPTIONS } from "@features/osquery/policies/model";
import {
  PolicyRemediationStatus,
  PolicyResultStatus,
} from "@features/osquery/policies/query-results";
import type { OsqueryPolicyHostStatus } from "@lib/api";
import { formatRelative } from "@lib/utils";

const policyColumns: DataTableColumnDef<OsqueryPolicyHostStatus>[] = [
  {
    accessorKey: "policy_name",
    header: () => "Policy",
    cell: ({ row }) => (
      <TextLink to="/osquery/policies/$id" params={{ id: String(row.original.policy_id) }}>
        {row.original.policy_name}
      </TextLink>
    ),
  },
  {
    accessorKey: "status",
    header: () => "Status",
    enableColumnFilter: true,
    cell: ({ row }) => <PolicyResultStatus status={row.original.status} />,
  },
  {
    id: "remediation",
    header: () => "Remediation",
    cell: ({ row }) =>
      row.original.remediation ? (
        <PolicyRemediationStatus status={row.original.remediation.status} />
      ) : (
        "-"
      ),
  },
  {
    accessorKey: "updated_at",
    header: () => "Last Evaluated",
    cell: ({ row }) => (row.original.updated_at ? formatRelative(row.original.updated_at) : "-"),
  },
];

const STATUS_FILTER_KEYS = [{ id: "status", multiple: true }] as const;
const routeApi = getRouteApi("/_authenticated/hosts/$id/policies");

function HostPoliciesToolbar({ table }: { table: DataTableInstance<OsqueryPolicyHostStatus> }) {
  return (
    <DataTableFacetedFilter
      column={table.getColumn("status")}
      title="Status"
      options={POLICY_RESULT_STATUS_OPTIONS}
    />
  );
}

export function HostOsqueryPoliciesTab({ hostId }: { hostId: number | null }) {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: STATUS_FILTER_KEYS,
  });
  const status = search.status;
  const query = useHostOsqueryPolicies(hostId, {
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
    columns: policyColumns,
    pageCount,
    rowCount: totalCount,
    getRowId: (row) => String(row.policy_id),
  });

  if (hostId === null) {
    return (
      <QueryError title="Failed to load policies" error={{ message: "Host route is invalid." }} />
    );
  }

  if (query.error) {
    return (
      <QueryError
        title="Failed to load policies"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (query.isLoading) {
    return <DataTableSkeleton columnCount={4} filterCount={1} />;
  }

  return (
    <DataTable
      table={table}
      pending={query.isPlaceholderData}
      heading="Policies"
      empty={
        <PanelEmptyState>
          {tableSearch.isFiltered ? "No matching policies" : "No policies yet"}
        </PanelEmptyState>
      }
    >
      <DataTableSearchInput
        loading={query.isPlaceholderData}
        value={tableSearch.q ?? ""}
        onValueChange={tableSearch.onQueryChange}
        placeholder="Search policies"
      />
      <HostPoliciesToolbar table={table} />
    </DataTable>
  );
}
