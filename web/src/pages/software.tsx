import { Link } from "@tanstack/react-router";
import { Package, Search, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

import { EmptyState } from "@/components/feedback/empty-state";
import { ErrorState } from "@/components/feedback/error-state";
import { Spinner } from "@/components/feedback/spinner";
import { PageHeader } from "@/components/layout/page-header";
import { FilterPopover, type FilterGroup } from "@/components/lists/filter-popover";
import type { SortState } from "@/components/lists/sort-state";
import { SortableTableHead } from "@/components/lists/sortable-table-head";
import { TablePagination } from "@/components/lists/table-pagination";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHeader, TableRow } from "@/components/ui/table";
import { useSoftware, type SoftwareTitle } from "@/hooks/use-software";
import { softwareSourceLabel, SOURCE_FILTER_OPTIONS } from "@/lib/software-source-labels";

const SEARCH_DEBOUNCE_MS = 200;
const DEFAULT_PAGE_SIZE = 50;

export interface SoftwareSearch {
  q?: string;
  source?: string[];
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: "asc" | "desc";
}

export function SoftwarePage({
  search,
  setSearch,
}: {
  search: SoftwareSearch;
  setSearch: (updater: (prev: SoftwareSearch) => SoftwareSearch) => void;
}) {

  const activeQuery = search.q ?? "";
  const activeSources = useMemo(() => search.source ?? [], [search.source]);
  const activePage = search.page ?? 1;
  const activePerPage = search.per_page ?? DEFAULT_PAGE_SIZE;
  const activeSort: SortState = {
    orderKey: search.order_key,
    orderDirection: search.order_direction,
  };
  const [searchInput, setSearchInput] = useState(activeQuery);
  const [lastActiveQuery, setLastActiveQuery] = useState(activeQuery);

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  if (lastActiveQuery !== activeQuery) {
    setLastActiveQuery(activeQuery);
    setSearchInput(activeQuery);
  }

  useEffect(
    () => () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    },
    [],
  );

  const writeQuery = (next: string) => {
    const trimmed = next.trim();
    setSearch((prev) => ({ ...prev, q: trimmed === "" ? undefined : trimmed, page: undefined }));
  };

  const handleSearchChange = (value: string) => {
    setSearchInput(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => writeQuery(value), SEARCH_DEBOUNCE_MS);
  };

  const handleSearchSubmit = () => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    writeQuery(searchInput);
  };

  const clearSearch = () => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    setSearchInput("");
    writeQuery("");
  };

  const setSources = (sources: string[]) => {
    setSearch((prev) => ({ ...prev, source: sources.length === 0 ? undefined : sources, page: undefined }));
  };

  const setPage = (page: number) => {
    setSearch((prev) => ({ ...prev, page: page <= 1 ? undefined : page }));
  };

  const setPerPage = (perPage: number) => {
    setSearch((prev) => ({
      ...prev,
      per_page: perPage === DEFAULT_PAGE_SIZE ? undefined : perPage,
      page: undefined,
    }));
  };

  const query = useSoftware({
    q: activeQuery,
    source: activeSources,
    page: activePage - 1,
    per_page: activePerPage,
    order_key: activeSort.orderKey,
    order_direction: activeSort.orderDirection,
  });
  const titles = useMemo(() => query.data?.items ?? [], [query.data?.items]);

  const totalCount = query.data?.count ?? titles.length;
  const hasFilters = activeQuery !== "" || activeSources.length > 0;
  const filterGroups: FilterGroup[] = [
    {
      id: "type",
      label: "Type",
      options: SOURCE_FILTER_OPTIONS,
      selected: activeSources,
      onChange: setSources,
    },
  ];

  return (
    <div className="flex flex-col">
      <PageHeader title="Software" description="Apps and packages discovered across enrolled hosts." />

      <div className="flex flex-col gap-4 p-6">
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-2">
            <div className="relative flex-1 max-w-md">
              <Search
                className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                aria-hidden
              />
              <Input
                value={searchInput}
                onChange={(event) => handleSearchChange(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter") handleSearchSubmit();
                }}
                placeholder="Search by name, display name, bundle id…"
                className="pl-8 pr-8"
                aria-label="Search software"
              />
              {searchInput ? (
                <button
                  type="button"
                  onClick={clearSearch}
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
                  {totalCount} title{totalCount === 1 ? "" : "s"}
                </>
              )}
            </div>
            <FilterPopover groups={filterGroups} />
          </div>
        </div>

        <SoftwareTable
          titles={titles}
          isLoading={query.isLoading}
          error={query.error}
          onRetry={() => query.refetch()}
          hasFilters={hasFilters}
          page={activePage}
          perPage={activePerPage}
          totalCount={totalCount}
          sort={activeSort}
          onSortChange={(next) => {
            setSearch((prev) => ({
              ...prev,
              order_key: next.orderKey,
              order_direction: next.orderDirection,
              page: undefined,
            }));
          }}
          onPageChange={setPage}
          onPerPageChange={setPerPage}
        />
      </div>
    </div>
  );
}

interface SoftwareTableProps {
  titles: SoftwareTitle[];
  isLoading: boolean;
  error: ReturnType<typeof useSoftware>["error"];
  onRetry: () => void;
  hasFilters: boolean;
  page: number;
  perPage: number;
  totalCount: number;
  sort: SortState;
  onSortChange: (next: SortState) => void;
  onPageChange: (page: number) => void;
  onPerPageChange: (perPage: number) => void;
}

function SoftwareTable({
  titles,
  isLoading,
  error,
  onRetry,
  hasFilters,
  page,
  perPage,
  totalCount,
  sort,
  onSortChange,
  onPageChange,
  onPerPageChange,
}: SoftwareTableProps) {
  if (error) {
    return <ErrorState message={error.message} onRetry={onRetry} />;
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Spinner /> Loading…
      </div>
    );
  }

  if (titles.length === 0) {
    return (
      <EmptyState
        icon={Package}
        title={hasFilters ? "No matches" : "No software inventory yet"}
        description={
          hasFilters
            ? "No titles matched the current filters."
            : "Hosts will report installed apps and packages on their next detail refresh."
        }
      />
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <SortableTableHead orderKey="name" active={sort} onSort={onSortChange}>
              Name
            </SortableTableHead>
            <SortableTableHead orderKey="versions_count" active={sort} onSort={onSortChange}>
              Version
            </SortableTableHead>
            <SortableTableHead orderKey="source" active={sort} onSort={onSortChange}>
              Type
            </SortableTableHead>
            <SortableTableHead orderKey="hosts_count" active={sort} onSort={onSortChange} align="right">
              Hosts
            </SortableTableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {titles.map((title) => (
            <SoftwareRow key={title.id} title={title} />
          ))}
        </TableBody>
      </Table>
      <TablePagination
        page={page}
        perPage={perPage}
        totalCount={totalCount}
        visibleCount={titles.length}
        onPageChange={onPageChange}
        onPerPageChange={onPerPageChange}
      />
    </div>
  );
}

function SoftwareRow({ title }: { title: SoftwareTitle }) {
  const versions = title.versions ?? [];
  const displayName = title.display_name || title.name;
  const versionLabel =
    versions.length === 0 ? "-" : versions.length === 1 ? versions[0].version || "-" : `${versions.length} versions`;

  return (
    <TableRow>
      <TableCell className="font-medium">
        <div className="min-w-0">
          <Link
            to="/software/titles/$softwareId"
            params={{ softwareId: title.id }}
            className="block truncate hover:underline"
            title={displayName}
          >
            {displayName}
          </Link>
        </div>
      </TableCell>
      <TableCell className="text-muted-foreground tabular-nums">{versionLabel}</TableCell>
      <TableCell className="text-muted-foreground" title={title.source}>
        {softwareSourceLabel(title.source, title.extension_for)}
      </TableCell>
      <TableCell className="text-right tabular-nums">{title.hosts_count}</TableCell>
    </TableRow>
  );
}
