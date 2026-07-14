import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { ServerCog } from "lucide-react";
import { parseAsInteger, useQueryStates } from "nuqs";
import * as React from "react";

import { BulkDeleteActionBar } from "@/components/bulk-delete-action-bar";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { DataTableViewOptions } from "@/components/data-table/data-table-view-options";
import { selectColumn } from "@/components/data-table/select-column";
import { FilterChip } from "@/components/filter-controls";
import { HostStatus } from "@/components/hosts/host-status";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { useAuth } from "@/hooks/use-auth";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { useBulkDeleteHosts, useHosts } from "@/hooks/use-hosts";
import { useSoftwareTitle } from "@/hooks/use-software";
import type { Host, SoftwareTitle } from "@/lib/api";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { formatBytes, formatRelative } from "@/lib/utils";

const STATUS_OPTIONS = [
  { value: "online", label: "Online" },
  { value: "offline", label: "Offline" },
] satisfies { value: Host["status"]; label: string }[];

const STATUS_FILTER_KEYS = [{ id: "status" }] as const;

// Deep-link filters set by other pages. Read and cleared via nuqs so chip
// removal stays reactive and does not clobber the nuqs-owned table state; the
// hosts route declares them in validateSearch for type-safe cross-page Links.
const deepLinkParsers = {
  label_id: parseAsInteger,
  software_title_id: parseAsInteger,
  software_id: parseAsInteger,
};

export function HostListPage() {
  const tableSearch = useDataTableSearch(STATUS_FILTER_KEYS);
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deepLink, setDeepLink] = useQueryStates(deepLinkParsers);

  const softwareTitle = useSoftwareTitle(deepLink.software_title_id);
  const softwareLabel = softwareFilterLabel({
    title: softwareTitle.data,
    softwareID: deepLink.software_id ?? undefined,
    softwareTitleID: deepLink.software_title_id ?? undefined,
  });

  const status = parseHostStatus(tableSearch.filters.status?.[0]);

  const query = useHosts({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    status,
    label_id: deepLink.label_id ?? undefined,
    software_title_id: deepLink.software_title_id ?? undefined,
    software_id: deepLink.software_id ?? undefined,
  });

  const hosts = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters =
    !!tableSearch.q ||
    !!status ||
    deepLink.label_id != null ||
    deepLink.software_id != null ||
    deepLink.software_title_id != null;

  const columns = React.useMemo<ColumnDef<Host>[]>(() => hostColumns, []);

  const table = useDataTable({
    tableState: tableSearch,
    data: hosts,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
    enableRowSelection: isAdmin,
  });

  return (
    <PageShell>
      <PageHeader
        title="Hosts"
        description="Track enrolled hosts, inventory, checks, reports, and Santa state."
        context={
          softwareLabel ? (
            <FilterChip
              label="Software"
              value={softwareLabel}
              onRemove={() => void setDeepLink({ software_id: null, software_title_id: null })}
            />
          ) : null
        }
      />

      {query.error ? (
        <QueryError
          title="Failed to load hosts"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={8} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          actionBar={
            isAdmin ? (
              <BulkDeleteActionBar
                table={table}
                useBulkDelete={useBulkDeleteHosts}
                noun="host"
                description="Agents can re-enroll with a valid Orbit secret."
              />
            ) : undefined
          }
          empty={
            <DataTableEmpty
              icon={<ServerCog />}
              filtered={hasFilters}
              title="No enrolled devices"
              description="Create an Orbit enrollment, then install the package on a host."
              filteredDescription="No hosts matched the current filters."
            />
          }
        >
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput className="h-8 w-40 lg:w-56" />
              <DataTableFacetedFilter
                column={table.getColumn("status")}
                title="Status"
                options={STATUS_OPTIONS}
              />
            </div>
            <div className="flex items-center gap-2">
              <DataTableViewOptions table={table} align="end" />
            </div>
          </div>
        </DataTable>
      )}
    </PageShell>
  );
}

function parseHostStatus(value: string | undefined): Host["status"] | undefined {
  return value === "online" || value === "offline" ? value : undefined;
}

const hostColumns: ColumnDef<Host>[] = [
  selectColumn<Host>(),
  {
    id: "display_name",
    accessorFn: (row) => row.display_name,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
    cell: ({ row }) => (
      <Link
        to="/hosts/$hostId"
        params={{ hostId: String(row.original.id) }}
        className="font-medium hover:underline"
      >
        {row.original.display_name}
      </Link>
    ),
    enableHiding: false,
    meta: { label: "Name" },
  },
  {
    id: "status",
    accessorFn: (row) => row.status,
    header: () => "Status",
    enableSorting: false,
    cell: ({ row }) => <HostStatus status={row.original.status} />,
    meta: { label: "Status", options: STATUS_OPTIONS },
    enableColumnFilter: true,
  },
  {
    id: "os.version",
    accessorFn: (row) => row.os.version,
    header: ({ column }) => <DataTableColumnHeader column={column} label="OS" />,
    cell: ({ row }) => row.original.os.version || "-",
    meta: { label: "OS" },
  },
  {
    id: "hardware.model_identifier",
    accessorFn: (row) => row.hardware.model_identifier,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Model" />,
    cell: ({ row }) => row.original.hardware.model_identifier || "-",
    meta: { label: "Model" },
  },
  {
    id: "hardware.serial",
    accessorFn: (row) => row.hardware.serial,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Serial" />,
    cell: ({ row }) => row.original.hardware.serial || "-",
    meta: { label: "Serial" },
  },
  {
    id: "storage.boot_volume.available_bytes",
    accessorFn: (row) => row.storage.boot_volume.available_bytes,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Disk Free" />,
    cell: ({ row }) =>
      row.original.storage.boot_volume.available_bytes
        ? formatBytes(row.original.storage.boot_volume.available_bytes)
        : "-",
    meta: { label: "Disk Free" },
  },
  {
    id: "primary_user",
    header: () => "User Email",
    enableSorting: false,
    cell: ({ row }) => row.original.primary_user?.email ?? "",
    meta: { label: "User Email" },
  },
  {
    id: "timestamps.last_seen_at",
    accessorFn: (row) => row.timestamps.last_seen_at,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Last Seen" />,
    cell: ({ row }) => formatRelative(row.original.timestamps.last_seen_at),
    meta: { label: "Last Seen" },
  },
  {
    id: "hardware.uuid",
    accessorFn: (row) => row.hardware.uuid,
    header: ({ column }) => <DataTableColumnHeader column={column} label="UUID" />,
    cell: ({ row }) => row.original.hardware.uuid || "-",
    meta: { label: "UUID" },
  },
  {
    id: "network.primary_ip",
    accessorFn: (row) => row.network.primary_ip,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Private IP" />,
    cell: ({ row }) => row.original.network.primary_ip ?? "-",
    meta: { label: "Private IP" },
  },
  {
    id: "network.last_remote_ip",
    accessorFn: (row) => row.network.last_remote_ip,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Public IP" />,
    cell: ({ row }) => row.original.network.last_remote_ip ?? "-",
    meta: { label: "Public IP" },
  },
  {
    id: "hardware.memory_bytes",
    accessorFn: (row) => row.hardware.memory_bytes,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Memory" />,
    cell: ({ row }) =>
      row.original.hardware.memory_bytes > 0
        ? formatBytes(row.original.hardware.memory_bytes)
        : "-",
    meta: { label: "Memory" },
  },
  {
    id: "agents.osquery.version",
    accessorFn: (row) => row.agents.osquery.version,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Osquery" />,
    cell: ({ row }) => row.original.agents.osquery.version || "-",
    meta: { label: "Osquery Version" },
  },
  {
    id: "timestamps.last_restarted_at",
    accessorFn: (row) => row.timestamps.last_restarted_at,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Last Restarted" />,
    cell: ({ row }) =>
      row.original.timestamps.last_restarted_at
        ? formatRelative(row.original.timestamps.last_restarted_at)
        : "-",
    meta: { label: "Last Restarted" },
  },
];

function softwareFilterLabel({
  title,
  softwareID,
  softwareTitleID,
}: {
  title: SoftwareTitle | undefined;
  softwareID: number | undefined;
  softwareTitleID: number | undefined;
}) {
  if (softwareID === undefined && softwareTitleID === undefined) return undefined;
  const titleName = title?.name;
  if (softwareID !== undefined && titleName) return `${titleName} version`;
  if (titleName) return titleName;
  if (softwareID !== undefined) return `Version #${softwareID}`;
  return `Title #${softwareTitleID}`;
}
