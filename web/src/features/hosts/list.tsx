import { getRouteApi } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { filesize } from "filesize";
import { ServerCog } from "lucide-react";
import * as React from "react";

import { BulkDeleteActionBar } from "@components/bulk-delete-action-bar";
import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { DataTableViewOptions } from "@components/data-table/data-table-view-options";
import { selectColumn } from "@components/data-table/select-column";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { FilterChip } from "@components/filter-controls";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { useAuth } from "@features/auth/queries";
import { HostStatus } from "@features/hosts/components/host-status";
import { listAllHosts, useBulkDeleteHosts, useHosts } from "@features/hosts/queries";
import { useLabel } from "@features/labels/queries";
import { useSoftwareTitle } from "@features/software/queries";
import type { Host, SoftwareTitle } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { formatRelative } from "@lib/utils";

const STATUS_OPTIONS = [
  { value: "online", label: "Online" },
  { value: "offline", label: "Offline" },
] satisfies { value: Host["status"]; label: string }[];

const STATUS_FILTER_KEYS = [{ id: "status" }] as const;
const routeApi = getRouteApi("/_authenticated/hosts/");

export function HostListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: STATUS_FILTER_KEYS,
    scopeKeys: ["label_id", "software_title_id", "software_id"],
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";

  const label = useLabel(search.label_id ?? null);
  const softwareTitle = useSoftwareTitle(search.software_title_id ?? null);
  const softwareLabel = softwareFilterLabel({
    title: softwareTitle.data,
    softwareID: search.software_id,
    softwareTitleID: search.software_title_id,
  });

  const query = useHosts(
    {
      q: tableSearch.q,
      page: tableSearch.page,
      per_page: tableSearch.per_page,
      sort: tableSearch.sort,
      status: search.status,
      label_id: search.label_id,
      software_title_id: search.software_title_id,
      software_id: search.software_id,
    },
    { refetchInterval: 30_000 },
  );

  const hosts = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const columns = React.useMemo<ColumnDef<Host>[]>(
    () => (isAdmin ? hostColumns : hostViewerColumns),
    [isAdmin],
  );

  const table = useDataTable({
    tableState: tableSearch,
    data: hosts,
    columns,
    pageCount,
    rowCount: totalCount,
    initialState: {
      pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE },
      columnVisibility: { "hardware.uuid": false },
    },
    getRowId: (row) => String(row.id),
    enableRowSelection: isAdmin,
  });
  const exportOptions: DataTableExportOptions<Host> = {
    filename: "hosts",
    columns: hostExportColumns,
    loadRows: () =>
      listAllHosts({
        q: tableSearch.q,
        sort: tableSearch.sort,
        status: search.status,
        label_id: search.label_id,
        software_title_id: search.software_title_id,
        software_id: search.software_id,
      }),
  };

  return (
    <PageShell>
      <PageHeader
        title="Hosts"
        description="Track enrolled hosts, inventory, checks, reports, and Santa state."
        context={
          <>
            {search.label_id !== undefined ? (
              <FilterChip
                label="Label"
                value={label.data?.name ?? `#${search.label_id}`}
                onRemove={() => tableSearch.clearSearchKeys(["label_id"])}
              />
            ) : null}
            {softwareLabel ? (
              <FilterChip
                label="Software"
                value={softwareLabel}
                onRemove={() => tableSearch.clearSearchKeys(["software_id", "software_title_id"])}
              />
            ) : null}
          </>
        }
      />

      {query.error ? (
        <QueryError
          title="Failed to load hosts"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={8} filterCount={1} withExport />
      ) : (
        <DataTable
          table={table}
          exportOptions={exportOptions}
          toolbarActions={<DataTableViewOptions table={table} align="end" />}
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
              filtered={tableSearch.isFiltered}
              title="No enrolled devices"
              description="Create an Orbit enrollment, then install the package on a host."
              filteredDescription="No hosts matched the current filters."
            />
          }
        >
          <div className="flex flex-wrap items-center gap-2">
            <DataTableSearchInput
              className="h-8 w-40 lg:w-56"
              value={tableSearch.q ?? ""}
              onValueChange={tableSearch.onQueryChange}
            />
            <DataTableFacetedFilter
              column={table.getColumn("status")}
              title="Status"
              options={STATUS_OPTIONS}
            />
          </div>
        </DataTable>
      )}
    </PageShell>
  );
}

const hostColumns: ColumnDef<Host>[] = [
  selectColumn<Host>(),
  {
    id: "display_name",
    accessorFn: (row) => row.display_name,
    header: "Name",
    cell: ({ row }) => (
      <Link to="/hosts/$id" params={{ id: String(row.original.id) }} className="font-medium">
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
    header: "OS",
    cell: ({ row }) => row.original.os.version || "-",
    meta: { label: "OS" },
  },
  {
    id: "hardware.model_identifier",
    accessorFn: (row) => row.hardware.model_identifier,
    header: "Model",
    cell: ({ row }) => row.original.hardware.model_identifier || "-",
    meta: { label: "Model" },
  },
  {
    id: "hardware.serial",
    accessorFn: (row) => row.hardware.serial,
    header: "Serial",
    cell: ({ row }) => row.original.hardware.serial || "-",
    meta: { label: "Serial" },
  },
  {
    id: "storage.boot_volume.available_bytes",
    accessorFn: (row) => row.storage.boot_volume.available_bytes,
    header: "Disk Free",
    cell: ({ row }) =>
      row.original.storage.boot_volume.available_bytes
        ? filesize(row.original.storage.boot_volume.available_bytes)
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
    header: "Last Seen",
    cell: ({ row }) => formatRelative(row.original.timestamps.last_seen_at),
    meta: { label: "Last Seen" },
  },
  {
    id: "hardware.uuid",
    accessorFn: (row) => row.hardware.uuid,
    header: "UUID",
    cell: ({ row }) => row.original.hardware.uuid || "-",
    meta: { label: "UUID" },
  },
  {
    id: "network.primary_ip",
    accessorFn: (row) => row.network.primary_ip,
    header: "Private IP",
    cell: ({ row }) => row.original.network.primary_ip ?? "-",
    meta: { label: "Private IP" },
  },
  {
    id: "network.last_remote_ip",
    accessorFn: (row) => row.network.last_remote_ip,
    header: "Public IP",
    cell: ({ row }) => row.original.network.last_remote_ip ?? "-",
    meta: { label: "Public IP" },
  },
  {
    id: "hardware.memory_bytes",
    accessorFn: (row) => row.hardware.memory_bytes,
    header: "Memory",
    cell: ({ row }) =>
      row.original.hardware.memory_bytes > 0 ? filesize(row.original.hardware.memory_bytes) : "-",
    meta: { label: "Memory" },
  },
  {
    id: "agents.osquery.version",
    accessorFn: (row) => row.agents.osquery.version,
    header: "Osquery",
    cell: ({ row }) => row.original.agents.osquery.version || "-",
    meta: { label: "Osquery Version" },
  },
  {
    id: "timestamps.last_restarted_at",
    accessorFn: (row) => row.timestamps.last_restarted_at,
    header: "Last Restarted",
    cell: ({ row }) =>
      row.original.timestamps.last_restarted_at
        ? formatRelative(row.original.timestamps.last_restarted_at)
        : "-",
    meta: { label: "Last Restarted" },
  },
];

const hostViewerColumns = hostColumns.filter((column) => column.id !== "select");

const hostExportColumns: DataTableExportOptions<Host>["columns"] = [
  { header: "Name", value: (host) => host.display_name },
  { header: "Status", value: (host) => host.status },
  { header: "OS", value: (host) => host.os.version },
  { header: "Model", value: (host) => host.hardware.model_identifier },
  { header: "Serial", value: (host) => host.hardware.serial },
  {
    header: "Disk Free",
    value: (host) =>
      host.storage.boot_volume.available_bytes
        ? filesize(host.storage.boot_volume.available_bytes)
        : "",
  },
  { header: "User Email", value: (host) => host.primary_user?.email },
  { header: "Last Seen", value: (host) => host.timestamps.last_seen_at },
  { header: "UUID", value: (host) => host.hardware.uuid },
  { header: "Private IP", value: (host) => host.network.primary_ip },
  { header: "Public IP", value: (host) => host.network.last_remote_ip },
  {
    header: "Memory",
    value: (host) => (host.hardware.memory_bytes > 0 ? filesize(host.hardware.memory_bytes) : ""),
  },
  { header: "Osquery Version", value: (host) => host.agents.osquery.version },
  {
    header: "Last Restarted",
    value: (host) => host.timestamps.last_restarted_at,
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
