import { Link, useNavigate, useSearch } from "@tanstack/react-router";
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
import { formatRelative } from "@/lib/utils";

const SEARCH_DEBOUNCE_MS = 200;
const DEFAULT_PAGE_SIZE = 50;
const PLATFORM_FILTER_OPTIONS = [{ value: "darwin", label: "Darwin" }];

export function HostsListPage() {
  const search = useSearch({ from: "/_authed/hosts/" });
  const navigate = useNavigate({ from: "/_authed/hosts/" });
  const activeQ = search.q ?? "";
  const activePage = search.page ?? 1;
  const activePerPage = search.per_page ?? DEFAULT_PAGE_SIZE;
  const activeSort: SortState = {
    orderKey: search.order_key,
    orderDirection: search.order_direction,
  };
  const activePlatform = search.platform ? [search.platform] : [];
  const [searchInput, setSearchInput] = useState(activeQ);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(
    () => () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    },
    [],
  );

  const writeQ = (next: string) => {
    const trimmed = next.trim();
    void navigate({
      to: "/hosts",
      search: (prev) => ({ ...prev, q: trimmed === "" ? undefined : trimmed, page: undefined }),
      replace: true,
    });
  };

  const query = useHosts({
    q: activeQ,
    page: activePage - 1,
    per_page: activePerPage,
    order_key: activeSort.orderKey,
    order_direction: activeSort.orderDirection,
    platform: search.platform,
    software_title_id: search.software_title_id,
    software_id: search.software_id,
  });
  const isSoftwareFiltered = Boolean(search.software_title_id || search.software_id);
  const hasFilters = activeQ !== "" || activePlatform.length > 0 || isSoftwareFiltered;
  const filterGroups: FilterGroup[] = [
    {
      id: "platform",
      label: "Platform",
      options: PLATFORM_FILTER_OPTIONS,
      selected: activePlatform,
      onChange: (values) => {
        void navigate({
          to: "/hosts",
          search: (prev) => ({ ...prev, platform: values[0] || undefined, page: undefined }),
          replace: true,
        });
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
              void navigate({
                to: "/hosts",
                search: (prev) => ({
                  ...prev,
                  order_key: next.orderKey,
                  order_direction: next.orderDirection,
                  page: undefined,
                }),
                replace: true,
              });
            }}
            onPageChange={(page) => {
              void navigate({
                to: "/hosts",
                search: (prev) => ({ ...prev, page: page <= 1 ? undefined : page }),
                replace: true,
              });
            }}
            onPerPageChange={(perPage) => {
              void navigate({
                to: "/hosts",
                search: (prev) => ({
                  ...prev,
                  per_page: perPage === DEFAULT_PAGE_SIZE ? undefined : perPage,
                  page: undefined,
                }),
                replace: true,
              });
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
              Name
            </SortableTableHead>
            <TableHead>Primary user</TableHead>
            <SortableTableHead orderKey="platform" active={sort} onSort={onSortChange}>
              Platform
            </SortableTableHead>
            <SortableTableHead orderKey="hardware_serial" active={sort} onSort={onSortChange}>
              Serial
            </SortableTableHead>
            <SortableTableHead orderKey="os_version" active={sort} onSort={onSortChange}>
              OS
            </SortableTableHead>
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
                <TableCell className="text-muted-foreground max-w-[16rem] truncate" title={primaryEmail || ""}>
                  {primaryEmail || "-"}
                </TableCell>
                <TableCell>
                  <PlatformBadge platform={row.platform} />
                </TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{row.hardware_serial || "-"}</TableCell>
                <TableCell className="text-muted-foreground">{row.os_version || "-"}</TableCell>
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

function PlatformBadge({ platform }: { platform: Host["platform"] }) {
  if (!platform) return <span className="text-muted-foreground">-</span>;
  return <Badge variant="muted">{platform}</Badge>;
}
