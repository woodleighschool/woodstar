import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { KeyRound, ServerCog, Trash2 } from "lucide-react";
import { useState } from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableColumnToggle } from "@/components/data-table/data-table-column-toggle";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { defaultHiddenIds } from "@/components/data-table/data-table-visibility";
import { PageActions } from "@/components/layout/page-actions";
import { OrbitEnrollSecretsDialog } from "@/components/secrets/orbit-enroll-secrets-dialog";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useHiddenColumns } from "@/hooks/use-hidden-columns";
import { useBulkDeleteHosts, useHosts, type Host } from "@/hooks/use-hosts";
import { useLabels } from "@/hooks/use-labels";
import { useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { PLATFORM_LABELS, QUERYABLE_PLATFORMS } from "@/lib/targeting";
import { formatBytes, formatRelative } from "@/lib/utils";

const PLATFORM_OPTIONS = QUERYABLE_PLATFORMS.map((platform) => ({ value: platform, label: PLATFORM_LABELS[platform] }));
const STATUS_OPTIONS = [
  { value: "online", label: "Online" },
  { value: "offline", label: "Offline" },
];

export function HostsListPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [selectedHostIds, setSelectedHostIds] = useState<string[]>([]);
  const labelsQuery = useLabels({
    per_page: 200,
    order_key: "name",
    order_direction: "asc",
    label_type: "regular",
  });
  const bulkDelete = useBulkDeleteHosts();

  const isSoftwareFiltered = !!search.software_title_id || !!search.software_id;

  const query = useHosts({
    q: search.q,
    page: state.page,
    per_page: state.perPage,
    order_key: state.orderKey,
    order_direction: state.orderDirection,
    status: search.status,
    platform: search.platform,
    label_id: search.label_id,
    software_title_id: search.software_title_id,
    software_id: search.software_id,
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  // Captured once on mount; "online" thresholds don't need second-by-second precision in a list.
  const [now] = useState(() => Date.now());

  const hasFilters = !!search.q || !!search.status || !!search.platform || !!search.label_id || isSoftwareFiltered;

  const allColumns: ColumnDef<Host>[] = [
    {
      id: "display_name",
      accessorFn: (row) => row.display_name || row.hardware_uuid,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) => row.original.display_name || row.original.hardware_uuid,
      meta: { label: "Host", alwaysVisible: true },
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
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.os_version || "-"}</span>,
      meta: { label: "OS" },
    },
    {
      id: "hardware_model",
      accessorKey: "hardware_model",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Model" />,
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.hardware_model || "-"}</span>,
      meta: { label: "Model" },
    },
    {
      id: "hardware_serial",
      accessorKey: "hardware_serial",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Serial" />,
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.hardware_serial || "-"}</span>,
      meta: { label: "Serial" },
    },
    {
      id: "disk_space_available_bytes",
      accessorKey: "disk_space_available_bytes",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Disk free" />,
      cell: ({ row }) => (
        <span className="text-muted-foreground tabular-nums">
          {row.original.disk_space_available_bytes ? formatBytes(row.original.disk_space_available_bytes) : "-"}
        </span>
      ),
      meta: { label: "Disk free" },
    },
    {
      id: "primary_user",
      header: () => "Primary user",
      enableSorting: false,
      cell: ({ row }) => {
        const email = row.original.device_mappings?.[0]?.email ?? "";
        return (
          <span className="text-muted-foreground block max-w-[16rem] truncate" title={email || ""}>
            {email || "-"}
          </span>
        );
      },
      meta: { label: "Primary user" },
    },
    {
      id: "last_seen_at",
      accessorKey: "last_seen_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Last seen" />,
      cell: ({ row }) => (
        <span
          className="text-muted-foreground"
          title={row.original.last_seen_at ? new Date(row.original.last_seen_at).toLocaleString() : ""}
        >
          {formatRelative(row.original.last_seen_at)}
        </span>
      ),
      meta: { label: "Last seen" },
    },
    {
      id: "hardware_uuid",
      accessorKey: "hardware_uuid",
      header: ({ column }) => <DataTableColumnHeader column={column} title="UUID" />,
      cell: ({ row }) => (
        <span className="text-muted-foreground font-mono text-xs">{row.original.hardware_uuid || "-"}</span>
      ),
      meta: { label: "UUID", defaultHidden: true },
    },
    {
      id: "primary_ip",
      accessorKey: "primary_ip",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Private IP" />,
      cell: ({ row }) => <span className="text-muted-foreground tabular-nums">{row.original.primary_ip ?? "-"}</span>,
      meta: { label: "Private IP", defaultHidden: true },
    },
    {
      id: "public_ip",
      accessorKey: "public_ip",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Public IP" />,
      cell: ({ row }) => <span className="text-muted-foreground tabular-nums">{row.original.public_ip ?? "-"}</span>,
      meta: { label: "Public IP", defaultHidden: true },
    },
    {
      id: "physical_memory",
      accessorKey: "physical_memory",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Memory" />,
      cell: ({ row }) => (
        <span className="text-muted-foreground tabular-nums">
          {row.original.physical_memory > 0 ? formatBytes(row.original.physical_memory) : "-"}
        </span>
      ),
      meta: { label: "Memory", defaultHidden: true },
    },
    {
      id: "osquery_version",
      accessorKey: "osquery_version",
      header: ({ column }) => <DataTableColumnHeader column={column} title="osquery" />,
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.osquery_version || "-"}</span>,
      meta: { label: "osquery version", defaultHidden: true },
    },
    {
      id: "last_restarted_at",
      accessorKey: "last_restarted_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Last restarted" />,
      cell: ({ row }) => (
        <span
          className="text-muted-foreground"
          title={row.original.last_restarted_at ? new Date(row.original.last_restarted_at).toLocaleString() : ""}
        >
          {row.original.last_restarted_at ? formatRelative(row.original.last_restarted_at) : "-"}
        </span>
      ),
      meta: { label: "Last restarted", defaultHidden: true },
    },
  ];

  const { hidden, toggle } = useHiddenColumns("hosts:hidden_columns", defaultHiddenIds(allColumns));
  const hiddenSet = new Set(hidden);
  const visibleColumns = allColumns.filter((c) => {
    if (c.meta?.alwaysVisible) return true;
    return !hiddenSet.has(c.id ?? "");
  });

  const labelOptions = (labelsQuery.data?.items ?? []).map((l) => ({ value: String(l.id), label: l.name }));
  const selectedIDs = selectedHostIds.map(Number);

  const deleteSelectedHosts = () => {
    const count = selectedIDs.length;
    if (count === 0) return;
    const confirmed = window.confirm(
      `Delete ${count} selected ${count === 1 ? "host" : "hosts"}? Deleted hosts will re-enroll if their agent can still use a valid enroll secret.`,
    );
    if (!confirmed) return;
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => setSelectedHostIds([]),
    });
  };

  return (
    <>
      <PageActions>
        {isSoftwareFiltered ? (
          <Button asChild variant="outline" size="sm">
            <Link to="/hosts">Clear filter</Link>
          </Button>
        ) : null}
        <OrbitEnrollSecretsDialog
          trigger={
            <Button variant="outline" size="sm" className="gap-2">
              <KeyRound data-icon="inline-start" /> Manage enroll secrets
            </Button>
          }
        />
      </PageActions>

      <div className="p-6">
        {query.error ? (
          <Alert variant="destructive">
            <AlertTitle>Failed to load hosts</AlertTitle>
            <AlertDescription>{query.error.message}</AlertDescription>
            <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
              Retry
            </Button>
          </Alert>
        ) : (
          <DataTable
            columns={visibleColumns}
            data={data}
            totalCount={totalCount}
            page={state.page}
            perPage={state.perPage}
            sort={{ orderKey: state.orderKey, orderDirection: state.orderDirection }}
            onPageChange={setters.setPage}
            onPerPageChange={setters.setPerPage}
            onSortChange={(s) => setters.setSort(s.orderKey, s.orderDirection)}
            isLoading={query.isLoading}
            enableRowSelection
            selectedRowIds={selectedHostIds}
            onSelectedRowIdsChange={setSelectedHostIds}
            bulkActions={
              <Button variant="destructive" size="sm" onClick={deleteSelectedHosts} disabled={bulkDelete.isPending}>
                <Trash2 data-icon="inline-start" />
                Delete
              </Button>
            }
            rowHref={(row) => ({ to: "/hosts/$hostId", params: { hostId: String(row.id) } })}
            toolbar={
              <HostsToolbar
                draft={draft}
                onDraftChange={setDraft}
                status={search.status}
                onStatusChange={(v) => setters.setFilter("status", v)}
                platform={search.platform}
                onPlatformChange={(v) => setters.setFilter("platform", v)}
                labelId={search.label_id}
                onLabelChange={(v) => setters.setFilter("label_id", v)}
                labelOptions={labelOptions}
                allColumns={allColumns}
                hiddenColumns={hidden}
                onToggleColumn={toggle}
                isFetching={query.isFetching}
                totalCount={totalCount}
              />
            }
            empty={
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <ServerCog />
                  </EmptyMedia>
                  <EmptyTitle>{hasFilters ? "No matches" : "No hosts enrolled yet"}</EmptyTitle>
                  <EmptyDescription>
                    {hasFilters
                      ? "No hosts matched the current filters."
                      : "Create an enroll secret, then point an Orbit-managed Mac at this Woodstar deployment."}
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            }
          />
        )}
      </div>
    </>
  );
}

interface HostsToolbarProps {
  draft: string;
  onDraftChange: (next: string) => void;
  status: string | undefined;
  onStatusChange: (next: string | undefined) => void;
  platform: string | undefined;
  onPlatformChange: (next: string | undefined) => void;
  labelId: string | undefined;
  onLabelChange: (next: string | undefined) => void;
  labelOptions: { value: string; label: string }[];
  allColumns: ColumnDef<Host>[];
  hiddenColumns: string[];
  onToggleColumn: (id: string) => void;
  isFetching: boolean;
  totalCount: number;
}

function HostsToolbar({
  draft,
  onDraftChange,
  status,
  onStatusChange,
  platform,
  onPlatformChange,
  labelId,
  onLabelChange,
  labelOptions,
  allColumns,
  hiddenColumns,
  onToggleColumn,
  isFetching,
  totalCount,
}: HostsToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <DataTableSearch value={draft} onChange={onDraftChange} placeholder="Search hosts" label="Search hosts" />
      <DataTableFacetedFilter
        title="Status"
        options={STATUS_OPTIONS}
        selected={status ? [status] : []}
        onChange={(next) => onStatusChange(next[0])}
        singleSelect
      />
      <DataTableFacetedFilter
        title="Platform"
        options={PLATFORM_OPTIONS}
        selected={platform ? [platform] : []}
        onChange={(next) => onPlatformChange(next[0])}
        singleSelect
      />
      <DataTableFacetedFilter
        title="Label"
        options={labelOptions}
        selected={labelId ? [labelId] : []}
        onChange={(next) => onLabelChange(next[0])}
        singleSelect
      />
      <DataTableColumnToggle columns={allColumns} hidden={hiddenColumns} onToggle={onToggleColumn} />
      <div className="text-muted-foreground ml-auto text-xs tabular-nums">
        {isFetching ? "Loading..." : `${totalCount} ${totalCount === 1 ? "host" : "hosts"}`}
      </div>
    </div>
  );
}

function HostStatusBadge({ host, now }: { host: Host; now: number }) {
  if (!host.last_seen_at) {
    return (
      <Badge variant="outline" className="gap-1.5">
        <span className="bg-status-offline size-2 rounded-full" />
        Offline
      </Badge>
    );
  }
  const lastSeen = new Date(host.last_seen_at).getTime();
  const online = now - lastSeen <= 5 * 60 * 1000;
  return (
    <Badge variant={online ? "default" : "outline"} className="gap-1.5">
      <span className={online ? "bg-status-online size-2 rounded-full" : "bg-status-offline size-2 rounded-full"} />
      {online ? "Online" : "Offline"}
    </Badge>
  );
}
