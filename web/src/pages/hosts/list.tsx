import { useSearch } from "@tanstack/react-router";
import type { ColumnDef, Table as TanStackTable } from "@tanstack/react-table";
import { CircleAlert, CircleCheck, ServerCog, Trash2 } from "lucide-react";
import { useState, type ReactNode } from "react";

import {
  BulkDeleteDialog,
  DataTable,
  DataTableColumnHeader,
  DataTableColumnToggle,
  DataTableEmptyState,
  DataTableFacetedFilter,
  DataTableSearch,
} from "@/components/data-table";
import { FilterChip, FilterSelect } from "@/components/filter-controls";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useCheck } from "@/hooks/use-checks";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useBulkDeleteHosts, useHosts, type Host } from "@/hooks/use-hosts";
import { useLabels } from "@/hooks/use-labels";
import { useSoftwareTitle, type SoftwareTitle } from "@/hooks/use-software";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { primaryDeviceMapping } from "@/lib/host-device-mappings";
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

export function HostsListPage() {
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
  const checkQuery = useCheck(checkID && checkResponse ? checkID : null);
  const softwareIDParam = typeof search.software_id === "string" ? search.software_id : undefined;
  const softwareTitleIDParam = typeof search.software_title_id === "string" ? search.software_title_id : undefined;
  const softwareTitleID = softwareTitleIDParam === undefined ? undefined : Number(softwareTitleIDParam);
  const softwareTitle = useSoftwareTitle(
    softwareTitleID !== undefined && Number.isFinite(softwareTitleID) ? softwareTitleID : null,
  );
  const isSoftwareFiltered = !!softwareTitleIDParam || !!softwareIDParam;
  const isCheckFiltered = checkID !== undefined || checkResponse !== undefined;

  const query = useHosts({
    q: search.q,
    ...tableQueryParams(state),
    status: search.status,
    label_id: search.label_id == null ? undefined : Number(search.label_id),
    software_title_id: softwareTitleID,
    software_id: softwareIDParam == null ? undefined : Number(softwareIDParam),
    check_id: checkID,
    check_response: checkResponse,
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  // Captured once on mount; "online" thresholds don't need second-by-second precision in a list.
  const [now] = useState(() => Date.now());

  const hasFilters = !!search.q || !!search.status || !!search.label_id || isSoftwareFiltered || isCheckFiltered;

  const allColumns: ColumnDef<Host>[] = [
    {
      id: "display_name",
      accessorFn: (row) => row.display_name || row.hardware_uuid,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => row.original.display_name || row.original.hardware_uuid,
      enableHiding: false,
      meta: { label: "Name" },
    },
    {
      id: "status",
      header: () => "Status",
      enableSorting: false,
      cell: ({ row }) => <HostStatusBadge host={row.original} now={now} />,
      meta: { label: "Status" },
    },
    {
      id: "os_version",
      accessorKey: "os_version",
      header: ({ column }) => <DataTableColumnHeader column={column} title="OS" />,
      cell: ({ row }) => row.original.os_version || "-",
      meta: { label: "OS" },
    },
    {
      id: "hardware_model",
      accessorKey: "hardware_model",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Model" />,
      cell: ({ row }) => row.original.hardware_model || "-",
      meta: { label: "Model" },
    },
    {
      id: "hardware_serial",
      accessorKey: "hardware_serial",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Serial" />,
      cell: ({ row }) => row.original.hardware_serial || "-",
      meta: { label: "Serial" },
    },
    {
      id: "disk_space_available_bytes",
      accessorKey: "disk_space_available_bytes",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Disk Free" />,
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.disk_space_available_bytes ? formatBytes(row.original.disk_space_available_bytes) : "-"}
        </span>
      ),
      meta: { label: "Disk Free" },
    },
    {
      id: "primary_user",
      header: () => "User",
      enableSorting: false,
      cell: ({ row }) => {
        const email = primaryDeviceMapping(row.original.device_mappings)?.email ?? "";
        return (
          <span className="block max-w-[16rem] truncate" title={email || ""}>
            {email || "-"}
          </span>
        );
      },
      meta: { label: "User" },
    },
    {
      id: "last_seen_at",
      accessorKey: "last_seen_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Last Seen" />,
      cell: ({ row }) => (
        <span title={row.original.last_seen_at ? new Date(row.original.last_seen_at).toLocaleString() : ""}>
          {formatRelative(row.original.last_seen_at)}
        </span>
      ),
      meta: { label: "Last Seen" },
    },
    {
      id: "hardware_uuid",
      accessorKey: "hardware_uuid",
      header: ({ column }) => <DataTableColumnHeader column={column} title="UUID" />,
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.hardware_uuid || "-"}</span>,
      meta: { label: "UUID" },
    },
    {
      id: "primary_ip",
      accessorKey: "primary_ip",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Private IP" />,
      cell: ({ row }) => <span className="tabular-nums">{row.original.primary_ip ?? "-"}</span>,
      meta: { label: "Private IP" },
    },
    {
      id: "public_ip",
      accessorKey: "public_ip",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Public IP" />,
      cell: ({ row }) => <span className="tabular-nums">{row.original.public_ip ?? "-"}</span>,
      meta: { label: "Public IP" },
    },
    {
      id: "physical_memory",
      accessorKey: "physical_memory",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Memory" />,
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.physical_memory > 0 ? formatBytes(row.original.physical_memory) : "-"}
        </span>
      ),
      meta: { label: "Memory" },
    },
    {
      id: "osquery_version",
      accessorKey: "osquery_version",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Osquery" />,
      cell: ({ row }) => row.original.osquery_version || "-",
      meta: { label: "Osquery Version" },
    },
    {
      id: "last_restarted_at",
      accessorKey: "last_restarted_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Last Restarted" />,
      cell: ({ row }) => (
        <span title={row.original.last_restarted_at ? new Date(row.original.last_restarted_at).toLocaleString() : ""}>
          {row.original.last_restarted_at ? formatRelative(row.original.last_restarted_at) : "-"}
        </span>
      ),
      meta: { label: "Last Restarted" },
    },
  ];

  const labelOptions = (labelsQuery.data?.items ?? []).map((l) => ({ value: String(l.id), label: l.name }));
  const selectedIDs = selectedHostIds.map(Number);

  const deleteSelectedHosts = () => {
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => {
        setSelectedHostIds([]);
        setDeleteOpen(false);
      },
    });
  };

  return (
    <PageShell>
      <PageHeader title="Hosts" description="Track enrolled hosts, inventory, checks, reports, and Santa state." />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Hosts</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
          <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
            Retry
          </Button>
        </Alert>
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
              checkId={checkID}
              checkName={checkQuery.data?.name}
              checkResponse={checkResponse}
              onCheckResponseChange={(v) => setters.setFilter("check_response", v)}
              onClearCheck={() => setters.setFilters({ check_id: undefined, check_response: undefined })}
              softwareLabel={softwareFilterLabel({
                title: softwareTitle.data,
                softwareID: softwareIDParam,
                softwareTitleID: softwareTitleIDParam,
              })}
              onClearSoftware={() => setters.setFilters({ software_id: undefined, software_title_id: undefined })}
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
  checkId: number | undefined;
  checkName: string | undefined;
  checkResponse: "pass" | "fail" | undefined;
  onCheckResponseChange: (next: string) => void;
  onClearCheck: () => void;
  softwareLabel: string | undefined;
  onClearSoftware: () => void;
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
  checkId,
  checkName,
  checkResponse,
  onCheckResponseChange,
  onClearCheck,
  softwareLabel,
  onClearSoftware,
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
      {checkId !== undefined ? (
        <FilterChip label="Check" value={checkName ?? `#${checkId}`} onRemove={onClearCheck} />
      ) : null}
      {checkId !== undefined && checkResponse ? (
        <FilterSelect
          label="Result"
          value={checkResponse}
          options={CHECK_RESPONSE_FILTERS}
          onChange={onCheckResponseChange}
        />
      ) : null}
      {softwareLabel ? <FilterChip label="Software" value={softwareLabel} onRemove={onClearSoftware} /> : null}
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
  softwareID: string | undefined;
  softwareTitleID: string | undefined;
}) {
  if (!softwareID && !softwareTitleID) return undefined;
  const titleName = title?.display_name ?? title?.name;
  if (softwareID && titleName) return `${titleName} version`;
  if (titleName) return titleName;
  if (softwareID) return `Version #${softwareID}`;
  return `Title #${softwareTitleID}`;
}

function HostStatusBadge({ host, now }: { host: Host; now: number }) {
  if (!host.last_seen_at) {
    return <Badge variant="secondary">Offline</Badge>;
  }
  const lastSeen = new Date(host.last_seen_at).getTime();
  const online = now - lastSeen <= 5 * 60 * 1000;
  return <Badge variant={online ? "success" : "secondary"}>{online ? "Online" : "Offline"}</Badge>;
}
