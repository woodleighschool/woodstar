import { getRouteApi } from "@tanstack/react-router";
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
import type { DataTableColumnDef } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { FilterChip } from "@components/filter-controls";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { RelativeTime } from "@components/relative-time";
import { useAuth } from "@features/auth/queries";
import { HostLastContact } from "@features/hosts/components/host-heartbeats";
import { HostOnlineDot } from "@features/hosts/components/host-online-dot";
import { listAllHosts, useBulkDeleteHosts, useHosts } from "@features/hosts/queries";
import { useLabel } from "@features/labels/queries";
import { useSoftwareTitle } from "@features/software/queries";
import type { Host, SoftwareTitle } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";

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
    scopeKeys: ["label_id", "software_title_id"],
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const softwareID = search.software_title_id === undefined ? undefined : search.software_id;

  const label = useLabel(search.label_id ?? null);
  const softwareTitle = useSoftwareTitle(search.software_title_id ?? null);
  const softwareLabel = softwareTitle.data
    ? softwareFilterLabel(softwareTitle.data, softwareID)
    : undefined;

  const query = useHosts(
    {
      q: tableSearch.q,
      page: tableSearch.page,
      per_page: tableSearch.per_page,
      sort: tableSearch.sort,
      status: search.status,
      label_id: search.label_id,
      software_title_id: search.software_title_id,
      software_id: softwareID,
    },
    { refetchInterval: 30_000 },
  );

  const hosts = React.useMemo(() => query.data?.items ?? [], [query.data?.items]);
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const columns = React.useMemo<DataTableColumnDef<Host>[]>(
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
      columnVisibility: { status: false, "hardware.uuid": false },
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
        software_id: softwareID,
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
        <DataTableSkeleton columnCount={8} filterCount={1} withExport withViewOptions />
      ) : (
        <DataTable
          table={table}
          pending={query.isPlaceholderData}
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
          <DataTableSearchInput
            loading={query.isPlaceholderData}
            value={tableSearch.q ?? ""}
            onValueChange={tableSearch.onQueryChange}
          />
          <DataTableFacetedFilter
            column={table.getColumn("status")}
            title="Status"
            options={STATUS_OPTIONS}
          />
        </DataTable>
      )}
    </PageShell>
  );
}

const hostColumns: DataTableColumnDef<Host>[] = [
  selectColumn<Host>(),
  {
    id: "display_name",
    accessorFn: (row) => row.display_name,
    header: "Name",
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <Link to="/hosts/$id" params={{ id: String(row.original.id) }} className="font-medium">
          {row.original.display_name}
        </Link>
        <HostOnlineDot status={row.original.status} />
      </div>
    ),
    enableHiding: false,
    size: 220,
    minSize: 140,
    meta: { label: "Name" },
  },
  {
    id: "status",
    accessorFn: (row) => row.status,
    enableHiding: false,
    enableSorting: false,
    meta: { options: STATUS_OPTIONS },
    enableColumnFilter: true,
  },
  {
    id: "os.version",
    accessorFn: (row) => row.os.version,
    header: "OS",
    cell: ({ row }) => row.original.os.version || "-",
    size: 96,
    meta: { label: "OS" },
  },
  {
    id: "hardware.model_identifier",
    accessorFn: (row) => row.hardware.model_identifier,
    header: "Model",
    cell: ({ row }) => row.original.hardware.model_identifier || "-",
    size: 160,
    meta: { label: "Model" },
  },
  {
    id: "hardware.serial",
    accessorFn: (row) => row.hardware.serial,
    header: "Serial",
    cell: ({ row }) => row.original.hardware.serial || "-",
    size: 136,
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
    size: 112,
    meta: { label: "Disk Free" },
  },
  {
    id: "primary_user",
    header: () => "User Email",
    enableSorting: false,
    cell: ({ row }) => row.original.primary_user?.email ?? "",
    size: 220,
    meta: { label: "User Email" },
  },
  {
    id: "last_contact",
    accessorFn: (row) => row.last_contact,
    header: "Last Contact",
    cell: ({ row }) => <HostLastContact host={row.original} />,
    size: 144,
    meta: { label: "Last Contact" },
  },
  {
    id: "hardware.uuid",
    accessorFn: (row) => row.hardware.uuid,
    header: "UUID",
    cell: ({ row }) => row.original.hardware.uuid || "-",
    size: 280,
    meta: { label: "UUID" },
  },
  {
    id: "network.primary_ip",
    accessorFn: (row) => row.network.primary_ip,
    header: "Private IP",
    cell: ({ row }) => row.original.network.primary_ip ?? "-",
    size: 176,
    meta: { label: "Private IP" },
  },
  {
    id: "public_ip",
    accessorFn: (row) => row.public_ip,
    header: "Public IP",
    cell: ({ row }) => row.original.public_ip ?? "-",
    size: 208,
    meta: { label: "Public IP" },
  },
  {
    id: "hardware.memory_bytes",
    accessorFn: (row) => row.hardware.memory_bytes,
    header: "Memory",
    cell: ({ row }) =>
      row.original.hardware.memory_bytes > 0 ? filesize(row.original.hardware.memory_bytes) : "-",
    size: 112,
    meta: { label: "Memory" },
  },
  {
    id: "agents.osquery.version",
    accessorFn: (row) => row.agents.osquery.version,
    header: "Osquery",
    cell: ({ row }) => row.original.agents.osquery.version || "-",
    size: 120,
    meta: { label: "Osquery Version" },
  },
  {
    id: "last_restarted_at",
    accessorFn: (row) => row.last_restarted_at,
    header: "Last Restarted",
    cell: ({ row }) =>
      row.original.last_restarted_at ? (
        <RelativeTime value={row.original.last_restarted_at} />
      ) : (
        "-"
      ),
    size: 152,
    meta: { label: "Last Restarted" },
  },
];

const hostViewerColumns = hostColumns.filter((column) => column.id !== "select");

const hostExportColumns: DataTableExportOptions<Host>["columns"] = [
  { header: "Name", value: (host) => host.display_name },
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
  { header: "Last Contact", value: (host) => host.last_contact },
  { header: "UUID", value: (host) => host.hardware.uuid },
  { header: "Private IP", value: (host) => host.network.primary_ip },
  { header: "Public IP", value: (host) => host.public_ip },
  {
    header: "Memory",
    value: (host) => (host.hardware.memory_bytes > 0 ? filesize(host.hardware.memory_bytes) : ""),
  },
  { header: "Osquery Version", value: (host) => host.agents.osquery.version },
  {
    header: "Last Restarted",
    value: (host) => host.last_restarted_at,
  },
];

function softwareFilterLabel(title: SoftwareTitle, softwareID: number | undefined) {
  if (softwareID === undefined) return title.name;
  const version = title.versions.items.find((item) => item.id === softwareID);
  if (!version) throw new Error(`Software version ${softwareID} is not in ${title.name}`);
  return `${title.name}: ${version.version}`;
}
