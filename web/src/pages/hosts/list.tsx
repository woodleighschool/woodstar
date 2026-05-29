import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef, Table as TanStackTable } from "@tanstack/react-table";
import { Check, ListFilter, ServerCog, Trash2 } from "lucide-react";
import { useState, type ReactNode } from "react";

import {
  BulkDeleteDialog,
  DataTable,
  DataTableColumnHeader,
  DataTableColumnToggle,
  DataTableEmptyState,
  DataTableSearch,
} from "@/components/data-table";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useBulkDeleteHosts, useHosts, type Host } from "@/hooks/use-hosts";
import { useLabels } from "@/hooks/use-labels";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { primaryDeviceMapping } from "@/lib/host-device-mappings";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { cn, formatBytes, formatRelative } from "@/lib/utils";

export function HostsListPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [selectedHostIds, setSelectedHostIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const labelsQuery = useLabels({
    page_size: MAX_PAGE_SIZE,
    sort: "name.asc",
    label_type: "regular",
  });
  const bulkDelete = useBulkDeleteHosts();

  const isSoftwareFiltered = !!search.software_title_id || !!search.software_id;

  const query = useHosts({
    q: search.q,
    ...tableQueryParams(state),
    status: search.status,
    label_id: search.label_id == null ? undefined : Number(search.label_id),
    software_title_id: search.software_title_id == null ? undefined : Number(search.software_title_id),
    software_id: search.software_id == null ? undefined : Number(search.software_id),
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  // Captured once on mount; "online" thresholds don't need second-by-second precision in a list.
  const [now] = useState(() => Date.now());

  const hasFilters = !!search.q || !!search.status || !!search.label_id || isSoftwareFiltered;

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
      <PageHeader
        title="Hosts"
        description="Track enrolled hosts, inventory, checks, reports, and Santa state."
        actions={
          isSoftwareFiltered ? (
            <Button asChild variant="outline" size="sm">
              <Link to="/hosts">Clear Filter</Link>
            </Button>
          ) : null
        }
      />

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
              labelId={search.label_id}
              onLabelChange={(v) => setters.setFilter("label_id", v)}
              labelOptions={labelOptions}
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
  labelId: string | undefined;
  onLabelChange: (next: string | undefined) => void;
  labelOptions: { value: string; label: string }[];
  table: TanStackTable<Host>;
  actions?: ReactNode;
}

function HostsToolbar({
  draft,
  onDraftChange,
  labelId,
  onLabelChange,
  labelOptions,
  table,
  actions,
}: HostsToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <DataTableSearch value={draft} onChange={onDraftChange} placeholder="Search" className="basis-full sm:basis-64" />
      <DataTableColumnToggle table={table} variant="ghost" />
      <HostFilterDropdown labelId={labelId} onLabelChange={onLabelChange} labelOptions={labelOptions} />
      {actions ? <div className="ml-auto">{actions}</div> : null}
    </div>
  );
}

interface HostFilterDropdownProps {
  labelId: string | undefined;
  onLabelChange: (next: string | undefined) => void;
  labelOptions: { value: string; label: string }[];
}

function HostFilterDropdown({ labelId, onLabelChange, labelOptions }: HostFilterDropdownProps) {
  const hasFilters = !!labelId;

  const clearFilters = () => {
    onLabelChange(undefined);
  };

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 border-dashed">
          <ListFilter data-icon="inline-start" />
          Filter by Label
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-72 p-0" align="end">
        <Command>
          <CommandInput placeholder="Search Labels" />
          <CommandList>
            <CommandEmpty>No Labels Found.</CommandEmpty>
            <CommandGroup heading="Labels">
              {labelOptions.map((option) => (
                <CommandItem
                  key={option.value}
                  value={`label ${option.label}`}
                  onSelect={() => onLabelChange(labelId === option.value ? undefined : option.value)}
                >
                  <SelectionCheck selected={labelId === option.value} />
                  <span>{option.label}</span>
                </CommandItem>
              ))}
            </CommandGroup>
            {hasFilters ? (
              <>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem onSelect={clearFilters} className="justify-center text-center">
                    Clear Filters
                  </CommandItem>
                </CommandGroup>
              </>
            ) : null}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

function SelectionCheck({ selected }: { selected: boolean }) {
  return (
    <div
      className={cn(
        "border-primary flex size-4 items-center justify-center rounded-sm border",
        selected ? "bg-primary text-primary-foreground" : "opacity-50 [&_svg]:invisible",
      )}
    >
      <Check />
    </div>
  );
}

function HostStatusBadge({ host, now }: { host: Host; now: number }) {
  if (!host.last_seen_at) {
    return <Badge variant="secondary">Offline</Badge>;
  }
  const lastSeen = new Date(host.last_seen_at).getTime();
  const online = now - lastSeen <= 5 * 60 * 1000;
  return <Badge variant={online ? "success" : "secondary"}>{online ? "Online" : "Offline"}</Badge>;
}
