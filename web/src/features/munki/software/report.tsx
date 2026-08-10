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
import type { MunkiSoftwareReportHost } from "@lib/api";
import { formatRelative } from "@lib/utils";

import { useMunkiSoftwareReport } from "./queries";

type SoftwareReportStatus = NonNullable<MunkiSoftwareReportHost["status"]>;

export const SOFTWARE_REPORT_STATUS_VALUES = ["installed", "pending"] as const;

const SOFTWARE_REPORT_STATUS_OPTIONS = [
  { label: "Installed", value: "installed" },
  { label: "Pending", value: "pending" },
] satisfies { label: string; value: SoftwareReportStatus }[];

const STATUS_FILTER_KEYS = [{ id: "status", multiple: true }] as const;
const routeApi = getRouteApi("/_authenticated/munki/software/$id/");

const reportColumns: DataTableColumnDef<MunkiSoftwareReportHost>[] = [
  {
    id: "host_name",
    accessorKey: "host_name",
    header: "Host",
    cell: ({ row }) => (
      <TextLink
        to="/hosts/$id"
        params={{ id: String(row.original.host_id) }}
        className="font-medium"
      >
        {row.original.host_name}
      </TextLink>
    ),
  },
  {
    id: "target_version",
    accessorKey: "target_version",
    header: "Target Version",
    cell: ({ row }) => row.original.target_version || "-",
  },
  {
    id: "status",
    accessorKey: "status",
    header: "Status",
    enableColumnFilter: true,
    cell: ({ row }) => softwareReportStatusLabel(row.original.status),
  },
  {
    id: "evaluated_at",
    accessorKey: "evaluated_at",
    header: "Last Evaluated",
    cell: ({ row }) =>
      row.original.evaluated_at ? formatRelative(row.original.evaluated_at) : "-",
  },
];

function SoftwareReportToolbar({ table }: { table: DataTableInstance<MunkiSoftwareReportHost> }) {
  return (
    <DataTableFacetedFilter
      column={table.getColumn("status")}
      title="Status"
      options={SOFTWARE_REPORT_STATUS_OPTIONS}
    />
  );
}

export function MunkiSoftwareReport({ softwareID }: { softwareID: number | null }) {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: STATUS_FILTER_KEYS,
  });
  const query = useMunkiSoftwareReport(softwareID, {
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    status: search.status,
  });
  const rows = useMemo(() => query.data?.items ?? [], [query.data?.items]);
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: rows,
    columns: reportColumns,
    pageCount,
    rowCount: totalCount,
    getRowId: (row) => String(row.host_id),
  });

  if (query.error) {
    return (
      <QueryError
        title="Failed to load software report"
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
      empty={
        <PanelEmptyState>
          {tableSearch.isFiltered ? "No matching software report rows" : "No expected hosts"}
        </PanelEmptyState>
      }
    >
      <DataTableSearchInput
        loading={query.isPlaceholderData}
        value={tableSearch.q ?? ""}
        onValueChange={tableSearch.onQueryChange}
        placeholder="Search hosts"
      />
      <SoftwareReportToolbar table={table} />
    </DataTable>
  );
}

function softwareReportStatusLabel(status: MunkiSoftwareReportHost["status"]) {
  switch (status) {
    case "installed":
      return "Installed";
    case "pending":
      return "Pending";
    default:
      return "-";
  }
}
