import { useSearch } from "@tanstack/react-router";
import type { ColumnDef, Table as TanStackTable } from "@tanstack/react-table";
import { CircleAlert, CircleCheck, ServerCog, Trash2 } from "lucide-react";
import { useState, type ReactNode } from "react";
import { toast } from "sonner";

import {
  BulkDeleteDialog,
  DataTable,
  DataTableColumnHeader,
  DataTableColumnToggle,
  DataTableEmptyState,
  DataTableFacetedFilter,
  DataTableSearch,
} from "@/components/data-table";
import { FilterChip } from "@/components/filter-controls";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useCheck } from "@/hooks/use-checks";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useBulkDeleteHosts, useHosts, type Host } from "@/hooks/use-hosts";
import { useLabels } from "@/hooks/use-labels";
import { useSoftwareTitle, type SoftwareTitle } from "@/hooks/use-software";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { formatBytes, formatRelative } from "@/lib/utils";

const HOST_STATUS_FILTERS = [
  { value: "online", label: "Online" },
  { value: "offline", label: "Offline" },
];

const CHECK_RESPONSE_FILTERS = [
  { value: "pass", label: "Pass", icon: CircleCheck },
  { value: "fail", label: "Fail", icon: CircleAlert },
];

export function HostListPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q", { resetKeys: ["page_index"] });
  const [selectedHostIds, setSelectedHostIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const labelsQuery = useLabels({
    page_size: MAX_PAGE_SIZE,
    sort: "name.asc",
    label_type: "regular",
  });
  const bulkDelete = useBulkDeleteHosts();

  const checkID = typeof search.check_id === "number" ? search.check_id : undefined;
  const checkResponse =
    search.check_response === "pass" || search.check_response === "fail" ? search.check_response : undefined;
  const checkQuery = useCheck(checkID ?? null);
  const softwareID = typeof search.software_id === "number" ? search.software_id : undefined;
  const softwareTitleID = typeof search.software_title_id === "number" ? search.software_title_id : undefined;
  const softwareTitle = useSoftwareTitle(softwareTitleID ?? null);
  const softwareLabel = softwareFilterLabel({
    title: softwareTitle.data,
    softwareID,
    softwareTitleID,
  });
  const isSoftwareFiltered = softwareTitleID !== undefined || softwareID !== undefined;
  const isCheckFiltered = checkID !== undefined || checkResponse !== undefined;

  const query = useHosts({
    q: search.q,
    ...tableQueryParams(state),
    status: search.status,
    label_id: search.label_id == null ? undefined : Number(search.label_id),
    software_title_id: softwareTitleID,
    software_id: softwareID,
    check_id: checkID,
    check_response: checkResponse,
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!search.status || !!search.label_id || isSoftwareFiltered || isCheckFiltered;

  const allColumns: ColumnDef<Host>[] = [
    {
      id: "display_name",
      accessorFn: (row) => row.display_name,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => row.original.display_name,
      enableHiding: false,
      meta: { label: "Name" },
    },
    {
      id: "status",
      header: () => "Status",
      enableSorting: false,
      cell: ({ row }) => <HostStatusBadge host={row.original} />,
      meta: { label: "Status" },
    },
    {
      id: "os.version",
      accessorFn: (row) => row.os.version,
      header: ({ column }) => <DataTableColumnHeader column={column} title="OS" />,
      cell: ({ row }) => row.original.os.version || "-",
      meta: { label: "OS" },
    },
    {
      id: "hardware.model_identifier",
      accessorFn: (row) => row.hardware.model_identifier,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Model" />,
      cell: ({ row }) => row.original.hardware.model_identifier || "-",
      meta: { label: "Model" },
    },
    {
      id: "hardware.serial",
      accessorFn: (row) => row.hardware.serial,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Serial" />,
      cell: ({ row }) => row.original.hardware.serial || "-",
      meta: { label: "Serial" },
    },
    {
      id: "storage.boot_volume.available_bytes",
      accessorFn: (row) => row.storage.boot_volume.available_bytes,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Disk Free" />,
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
      cell: ({ row }) => row.original.user_affinity.primary?.email ?? "",
      meta: { label: "User Email" },
    },
    {
      id: "timestamps.last_seen_at",
      accessorFn: (row) => row.timestamps.last_seen_at,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Last Seen" />,
      cell: ({ row }) => formatRelative(row.original.timestamps.last_seen_at),
      meta: { label: "Last Seen" },
    },
    {
      id: "hardware.uuid",
      accessorFn: (row) => row.hardware.uuid,
      header: ({ column }) => <DataTableColumnHeader column={column} title="UUID" />,
      cell: ({ row }) => row.original.hardware.uuid || "-",
      meta: { label: "UUID" },
    },
    {
      id: "network.primary_ip",
      accessorFn: (row) => row.network.primary_ip,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Private IP" />,
      cell: ({ row }) => row.original.network.primary_ip ?? "-",
      meta: { label: "Private IP" },
    },
    {
      id: "network.last_remote_ip",
      accessorFn: (row) => row.network.last_remote_ip,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Public IP" />,
      cell: ({ row }) => row.original.network.last_remote_ip ?? "-",
      meta: { label: "Public IP" },
    },
    {
      id: "hardware.memory_bytes",
      accessorFn: (row) => row.hardware.memory_bytes,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Memory" />,
      cell: ({ row }) =>
        row.original.hardware.memory_bytes > 0 ? formatBytes(row.original.hardware.memory_bytes) : "-",
      meta: { label: "Memory" },
    },
    {
      id: "agents.osquery.version",
      accessorFn: (row) => row.agents.osquery.version,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Osquery" />,
      cell: ({ row }) => row.original.agents.osquery.version || "-",
      meta: { label: "Osquery Version" },
    },
    {
      id: "timestamps.last_restarted_at",
      accessorFn: (row) => row.timestamps.last_restarted_at,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Last Restarted" />,
      cell: ({ row }) =>
        row.original.timestamps.last_restarted_at ? formatRelative(row.original.timestamps.last_restarted_at) : "-",
      meta: { label: "Last Restarted" },
    },
  ];

  const labelOptions = (labelsQuery.data?.items ?? []).map((l) => ({ value: String(l.id), label: l.name }));
  const selectedIDs = selectedHostIds.map(Number);

  const deleteSelectedHosts = () => {
    const count = selectedIDs.length;
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => {
        toast.success(`Deleted ${count} ${count === 1 ? "host" : "hosts"}`);
        setSelectedHostIds([]);
        setDeleteOpen(false);
      },
    });
  };

  return (
    <PageShell>
      <PageHeader
        title="Hosts"
        description="Track enrolled hosts, inventory, checks, reports, and Santa state."
        context={
          <>
            {checkID !== undefined ? (
              <FilterChip
                label="Check"
                value={checkQuery.data?.name ?? `#${checkID}`}
                onRemove={() => setters.setFilters({ check_id: undefined, check_response: undefined })}
              />
            ) : null}
            {softwareLabel ? (
              <FilterChip
                label="Software"
                value={softwareLabel}
                onRemove={() => setters.setFilters({ software_id: undefined, software_title_id: undefined })}
              />
            ) : null}
          </>
        }
      />

      {query.error ? (
        <QueryError title="Failed to load hosts" error={query.error} onRetry={() => void query.refetch()} />
      ) : (
        <DataTable
          columns={allColumns}
          data={data}
          totalCount={totalCount}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          showExport
          exportFilename="hosts.csv"
          enableRowSelection
          selectedRowIds={selectedHostIds}
          onSelectedRowIdsChange={setSelectedHostIds}
          bulkActions={
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)} disabled={bulkDelete.isPending}>
              <Trash2 data-icon="inline-start" />
              Delete
            </Button>
          }
          rowHref={(row) => ({ to: "/hosts/$hostId", params: { hostId: String(row.id) } })}
          toolbar={(table, exportButton) => (
            <HostsToolbar
              draft={draft}
              onDraftChange={setDraft}
              status={search.status}
              onStatusChange={(v) => setters.setFilter("status", v)}
              labelId={search.label_id}
              onLabelChange={(v) => setters.setFilter("label_id", v)}
              labelOptions={labelOptions}
              checkResponse={checkResponse}
              onCheckResponseChange={(v) => setters.setFilter("check_response", v)}
              hasCheckFilter={checkID !== undefined}
              table={table}
              actions={exportButton}
            />
          )}
          empty={
            <DataTableEmptyState
              icon={<ServerCog />}
              title={hasFilters ? "No Matches" : "No Enrolled Devices"}
              description={
                hasFilters ? (
                  "No hosts matched the current filters."
                ) : (
                  <>Create an Orbit enrollment, then install the package on a host.</>
                )
              }
            />
          }
        />
      )}
      <BulkDeleteDialog
        open={deleteOpen}
        onOpenChange={(open) => {
          if (!open) bulkDelete.reset();
          setDeleteOpen(open);
        }}
        count={selectedIDs.length}
        noun="host"
        description="Agents can re-enroll with a valid Orbit secret."
        pending={bulkDelete.isPending}
        onConfirm={deleteSelectedHosts}
      />
    </PageShell>
  );
}

interface HostsToolbarProps {
  draft: string;
  onDraftChange: (next: string) => void;
  status: string | undefined;
  onStatusChange: (next: string | undefined) => void;
  labelId: string | undefined;
  onLabelChange: (next: string | undefined) => void;
  labelOptions: { value: string; label: string }[];
  checkResponse: "pass" | "fail" | undefined;
  onCheckResponseChange: (next: string) => void;
  hasCheckFilter: boolean;
  table: TanStackTable<Host>;
  actions?: ReactNode;
}

function HostsToolbar({
  draft,
  onDraftChange,
  status,
  onStatusChange,
  labelId,
  onLabelChange,
  labelOptions,
  checkResponse,
  onCheckResponseChange,
  hasCheckFilter,
  table,
  actions,
}: HostsToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <DataTableSearch value={draft} onChange={onDraftChange} placeholder="Search" className="basis-full sm:basis-64" />
      <DataTableColumnToggle table={table} variant="ghost" />
      <DataTableFacetedFilter
        title="Status"
        options={HOST_STATUS_FILTERS}
        selected={status ? [status] : []}
        onChange={(next) => onStatusChange(next.at(0))}
        singleSelect
      />
      <DataTableFacetedFilter
        title="Label"
        options={labelOptions}
        selected={labelId ? [labelId] : []}
        onChange={(next) => onLabelChange(next.at(0))}
        singleSelect
      />
      {hasCheckFilter && checkResponse ? (
        <DataTableFacetedFilter
          title="Result"
          options={CHECK_RESPONSE_FILTERS}
          selected={[checkResponse]}
          onChange={(next) => onCheckResponseChange(next.at(0) ?? checkResponse)}
          singleSelect
        />
      ) : null}
      {actions ? <div className="ml-auto">{actions}</div> : null}
    </div>
  );
}

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
  const titleName = title?.display_name ?? title?.name;
  if (softwareID !== undefined && titleName) return `${titleName} version`;
  if (titleName) return titleName;
  if (softwareID !== undefined) return `Version #${softwareID}`;
  return `Title #${softwareTitleID}`;
}

function HostStatusBadge({ host }: { host: Host }) {
  const online = host.status === "online";
  return <Badge variant={online ? "success" : "secondary"}>{online ? "Online" : "Offline"}</Badge>;
}
