import { Link } from "@tanstack/react-router";
import { KeyRound, Search, ServerCog, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { EmptyState } from "@/components/feedback/empty-state";
import { ErrorState } from "@/components/feedback/error-state";
import { Spinner } from "@/components/feedback/spinner";
import { PageHeader } from "@/components/layout/page-header";
import { FilterPopover, type FilterGroup } from "@/components/lists/filter-popover";
import type { SortState } from "@/components/lists/sort-state";
import { SortableTableHead } from "@/components/lists/sortable-table-head";
import { TablePagination } from "@/components/lists/table-pagination";
import { OrbitEnrollSecretsDialog } from "@/components/secrets/orbit-enroll-secrets-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useHosts, type Host } from "@/hooks/use-hosts";
import { useLabels } from "@/hooks/use-labels";
import { formatBytes, formatRelative } from "@/lib/utils";

const SEARCH_DEBOUNCE_MS = 200;
const DEFAULT_PAGE_SIZE = 50;
const PLATFORM_FILTER_OPTIONS = [{ value: "darwin", label: "Darwin" }];
const STATUS_FILTER_OPTIONS = [
  { value: "online", label: "Online" },
  { value: "offline", label: "Offline" },
];

export interface HostsListSearch {
  q?: string;
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: "asc" | "desc";
  status?: string;
  platform?: string;
  label_id?: string;
  software_title_id?: string;
  software_id?: string;
}

export function HostsListPage({
  search,
  setSearch,
}: {
  search: HostsListSearch;
  setSearch: (updater: (prev: HostsListSearch) => HostsListSearch) => void;
}) {
  const activeQ = search.q ?? "";
  const activePage = search.page ?? 1;
  const activePerPage = search.per_page ?? DEFAULT_PAGE_SIZE;
  const activeSort: SortState = {
    orderKey: search.order_key,
    orderDirection: search.order_direction,
  };
  const activePlatform = search.platform ? [search.platform] : [];
  const activeStatus = search.status ? [search.status] : [];
  const activeLabel = search.label_id ? [search.label_id] : [];
  const [searchInput, setSearchInput] = useState(activeQ);
  const [lastActiveQ, setLastActiveQ] = useState(activeQ);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const labelsQuery = useLabels({ per_page: 200, order_key: "name", order_direction: "asc" });

  if (lastActiveQ !== activeQ) {
    setLastActiveQ(activeQ);
    setSearchInput(activeQ);
  }

  useEffect(
    () => () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    },
    [],
  );

  const writeQ = (next: string) => {
    const trimmed = next.trim();
    setSearch((prev) => ({ ...prev, q: trimmed === "" ? undefined : trimmed, page: undefined }));
  };

  const query = useHosts({
    q: activeQ,
    page: activePage - 1,
    per_page: activePerPage,
    order_key: activeSort.orderKey,
    order_direction: activeSort.orderDirection,
    status: search.status,
    platform: search.platform,
    label_id: search.label_id,
    software_title_id: search.software_title_id,
    software_id: search.software_id,
  });
  const isSoftwareFiltered = Boolean(search.software_title_id || search.software_id);
  const hasFilters =
    activeQ !== "" ||
    activePlatform.length > 0 ||
    activeStatus.length > 0 ||
    activeLabel.length > 0 ||
    isSoftwareFiltered;
  const filterGroups: FilterGroup[] = [
    {
      id: "status",
      label: "Status",
      options: STATUS_FILTER_OPTIONS,
      selected: activeStatus,
      onChange: (values) => {
        setSearch((prev) => ({ ...prev, status: values[0] || undefined, page: undefined }));
      },
    },
    {
      id: "platform",
      label: "Platform",
      options: PLATFORM_FILTER_OPTIONS,
      selected: activePlatform,
      onChange: (values) => {
        setSearch((prev) => ({ ...prev, platform: values[0] || undefined, page: undefined }));
      },
    },
    {
      id: "label",
      label: "Label",
      options: (labelsQuery.data?.items ?? []).map((label) => ({ value: label.id, label: label.name })),
      selected: activeLabel,
      onChange: (values) => {
        setSearch((prev) => ({ ...prev, label_id: values[0] || undefined, page: undefined }));
      },
    },
  ];

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Hosts"
        description={
          isSoftwareFiltered
            ? "Hosts matching the selected software filter."
            : "Orbit/osquery-managed Macs in this Woodstar deployment."
        }
        actions={
          <div className="flex items-center gap-2">
            {isSoftwareFiltered ? (
              <Button asChild variant="outline" size="sm">
                <Link to="/hosts">Clear filter</Link>
              </Button>
            ) : null}
            <OrbitEnrollSecretsDialog
              trigger={
                <Button variant="outline" size="sm" className="gap-2">
                  <KeyRound className="size-4" /> Manage enroll secrets
                </Button>
              }
            />
          </div>
        }
      />

      <div className="p-6">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <div className="relative flex-1 max-w-md">
              <Search
                className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                aria-hidden
              />
              <Input
                value={searchInput}
                onChange={(event) => {
                  setSearchInput(event.target.value);
                  if (debounceRef.current) clearTimeout(debounceRef.current);
                  debounceRef.current = setTimeout(() => writeQ(event.target.value), SEARCH_DEBOUNCE_MS);
                }}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    if (debounceRef.current) clearTimeout(debounceRef.current);
                    writeQ(searchInput);
                  }
                }}
                placeholder="Search hosts"
                className="pl-8 pr-8"
                aria-label="Search hosts"
              />
              {searchInput ? (
                <button
                  type="button"
                  onClick={() => {
                    if (debounceRef.current) clearTimeout(debounceRef.current);
                    setSearchInput("");
                    writeQ("");
                  }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted-foreground hover:text-foreground"
                  aria-label="Clear search"
                >
                  <X className="size-3.5" />
                </button>
              ) : null}
            </div>
            <div className="text-xs text-muted-foreground tabular-nums">
              {query.isFetching ? (
                <span className="inline-flex items-center gap-1">
                  <Spinner className="size-3" /> Loading…
                </span>
              ) : (
                <>
                  {query.data?.count ?? 0} host{query.data?.count === 1 ? "" : "s"}
                </>
              )}
            </div>
            <FilterPopover groups={filterGroups} />
          </div>

          <HostsTable
            query={query}
            hasFilters={hasFilters}
            page={activePage}
            perPage={activePerPage}
            sort={activeSort}
            onSortChange={(next) => {
              setSearch((prev) => ({
                ...prev,
                order_key: next.orderKey,
                order_direction: next.orderDirection,
                page: undefined,
              }));
            }}
            onPageChange={(page) => {
              setSearch((prev) => ({ ...prev, page: page <= 1 ? undefined : page }));
            }}
            onPerPageChange={(perPage) => {
              setSearch((prev) => ({
                ...prev,
                per_page: perPage === DEFAULT_PAGE_SIZE ? undefined : perPage,
                page: undefined,
              }));
            }}
          />
        </div>
      </div>
    </div>
  );
}

function HostsTable({
  query,
  hasFilters,
  page,
  perPage,
  sort,
  onSortChange,
  onPageChange,
  onPerPageChange,
}: {
  query: ReturnType<typeof useHosts>;
  hasFilters: boolean;
  page: number;
  perPage: number;
  sort: SortState;
  onSortChange: (next: SortState) => void;
  onPageChange: (page: number) => void;
  onPerPageChange: (perPage: number) => void;
}) {
  const [now] = useState(() => Date.now());

  if (query.error) {
    return <ErrorState message={query.error.message} onRetry={() => query.refetch()} />;
  }

  if (query.isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Spinner /> Loading…
      </div>
    );
  }

  const data = query.data?.items ?? [];
  if (data.length === 0) {
    return (
      <EmptyState
        icon={ServerCog}
        title={hasFilters ? "No matches" : "No hosts enrolled yet"}
        description={
          hasFilters
            ? "No hosts matched the current filters."
            : "Create an enroll secret, then point an Orbit-managed Mac at this Woodstar deployment. Hosts appear here on first check-in."
        }
      />
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <SortableTableHead orderKey="display_name" active={sort} onSort={onSortChange}>
              Host
            </SortableTableHead>
            <TableHead>Status</TableHead>
            <SortableTableHead orderKey="os_version" active={sort} onSort={onSortChange}>
              Operating system
            </SortableTableHead>
            <SortableTableHead orderKey="hardware_model" active={sort} onSort={onSortChange}>
              Hardware model
            </SortableTableHead>
            <SortableTableHead orderKey="hardware_serial" active={sort} onSort={onSortChange}>
              Serial number
            </SortableTableHead>
            <SortableTableHead orderKey="disk_space_available_bytes" active={sort} onSort={onSortChange}>
              Disk space
            </SortableTableHead>
            <TableHead>Primary user</TableHead>
            <SortableTableHead orderKey="last_seen_at" active={sort} onSort={onSortChange}>
              Last seen
            </SortableTableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row) => {
            const primaryEmail = row.device_mappings?.[0]?.email ?? "";
            return (
              <TableRow key={row.id}>
                <TableCell className="font-medium">
                  <Link to="/hosts/$hostId" params={{ hostId: row.id }} className="hover:underline">
                    {row.display_name || row.hardware_uuid}
                  </Link>
                </TableCell>
                <TableCell>
                  <StatusBadge host={row} now={now} />
                </TableCell>
                <TableCell className="text-muted-foreground">{row.os_version || "-"}</TableCell>
                <TableCell className="text-muted-foreground">{row.hardware_model || "-"}</TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{row.hardware_serial || "-"}</TableCell>
                <TableCell className="text-muted-foreground">
                  {row.disk_space_available_bytes ? formatBytes(row.disk_space_available_bytes) : "-"}
                </TableCell>
                <TableCell className="text-muted-foreground max-w-[16rem] truncate" title={primaryEmail || ""}>
                  {primaryEmail || "-"}
                </TableCell>
                <TableCell
                  className="text-muted-foreground"
                  title={row.last_seen_at ? new Date(row.last_seen_at).toLocaleString() : ""}
                >
                  {formatRelative(row.last_seen_at)}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
      <TablePagination
        page={page}
        perPage={perPage}
        totalCount={query.data?.count ?? data.length}
        visibleCount={data.length}
        onPageChange={onPageChange}
        onPerPageChange={onPerPageChange}
      />
    </div>
  );
}

function StatusBadge({ host, now }: { host: Host; now: number }) {
  if (!host.last_seen_at) return <Badge variant="outline">Offline</Badge>;
  const lastSeen = new Date(host.last_seen_at).getTime();
  const online = now - lastSeen <= 5 * 60 * 1000;
  return <Badge variant={online ? "default" : "outline"}>{online ? "Online" : "Offline"}</Badge>;
}
