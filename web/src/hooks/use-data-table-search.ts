import type {
  ColumnFiltersState,
  PaginationState,
  SortingState,
  Updater,
} from "@tanstack/react-table";
import * as React from "react";

import { useCallbackRef } from "@/hooks/use-callback-ref";
import { normalizePage, normalizePageSize } from "@/lib/pagination";
import { parseSortingState, serializeSortingState } from "@/lib/parsers";

// Encodes a single-column sort in the backend wire format.
export function encodeSort(id: string, desc = false): string {
  return serializeSortingState([{ id, desc }]);
}

interface DataTableSearchState {
  q?: string;
  page: number;
  per_page: number;
  sort?: string;
}

export interface DataTableFilterKey<Search extends DataTableSearchState> {
  id: Extract<keyof Search, string>;
  multiple?: boolean;
}

export interface DataTableQuery<Search extends DataTableSearchState = DataTableSearchState> {
  q?: string;
  page: number;
  per_page: number;
  sort?: string;
  // Column filter values are always arrays: useDataTable stores every column
  // filter as an array, so we parse them the same way (single-select facets just
  // carry a one-element array). Keys are sparse (only active filters present), so
  // the value is optional. Read `filters.x?.[0]` for single, `filters.x ?? []` for
  // multi.
  filters: Record<string, string[] | undefined>;
  isFiltered: boolean;
  pagination: PaginationState;
  sorting: SortingState;
  columnFilters: ColumnFiltersState;
  onPaginationChange: (updaterOrValue: Updater<PaginationState>) => void;
  onSortingChange: (updaterOrValue: Updater<SortingState>) => void;
  onColumnFiltersChange: (updaterOrValue: Updater<ColumnFiltersState>) => void;
  onQueryChange: (value: string | undefined) => void;
  clearSearchKeys: (keys: readonly Extract<keyof Search, string>[]) => void;
}

interface UseDataTableSearchOptions<Search extends DataTableSearchState> {
  search: Search;
  onSearchChange: (updater: (previous: Search) => Search) => void;
  filterKeys?: readonly DataTableFilterKey<Search>[];
  scopeKeys?: readonly Extract<keyof Search, string>[];
}

export function useDataTableSearch<Search extends DataTableSearchState>({
  search,
  onSearchChange,
  filterKeys = [],
  scopeKeys = [],
}: UseDataTableSearchOptions<Search>): DataTableQuery<Search> {
  const updateSearch = useCallbackRef(onSearchChange);
  const filters = React.useMemo(() => {
    const out: Record<string, string[] | undefined> = {};
    for (const { id } of filterKeys) {
      const value = search[id];
      if (Array.isArray(value)) {
        if (value.length > 0) out[id] = value.map(String);
      } else if (value != null && value !== "") {
        out[id] = [String(value)];
      }
    }
    return out;
  }, [filterKeys, search]);

  const pagination = React.useMemo<PaginationState>(
    () => ({
      pageIndex: search.page - 1,
      pageSize: search.per_page,
    }),
    [search.page, search.per_page],
  );

  const sorting = React.useMemo(() => parseSortingState(search.sort), [search.sort]);

  const columnFilters = React.useMemo<ColumnFiltersState>(
    () =>
      Object.entries(filters).map(([id, value]) => ({
        id,
        value: value ?? [],
      })),
    [filters],
  );

  const onPaginationChange = React.useCallback(
    (updaterOrValue: Updater<PaginationState>) => {
      const next =
        typeof updaterOrValue === "function" ? updaterOrValue(pagination) : updaterOrValue;
      updateSearch((previous) => ({
        ...previous,
        page: normalizePage(next.pageIndex + 1),
        per_page: normalizePageSize(next.pageSize),
      }));
    },
    [pagination, updateSearch],
  );

  const onSortingChange = React.useCallback(
    (updaterOrValue: Updater<SortingState>) => {
      const next = typeof updaterOrValue === "function" ? updaterOrValue(sorting) : updaterOrValue;
      const sort = serializeSortingState(singleSort(next)) || undefined;
      updateSearch((previous) => ({ ...previous, page: 1, sort }));
    },
    [sorting, updateSearch],
  );

  const onColumnFiltersChange = React.useCallback(
    (updaterOrValue: Updater<ColumnFiltersState>) => {
      const next =
        typeof updaterOrValue === "function" ? updaterOrValue(columnFilters) : updaterOrValue;
      updateSearch((previous) => {
        const values = Object.fromEntries(
          filterKeys.map(({ id, multiple }) => {
            const value = next.find((filter) => filter.id === id)?.value;
            const items = Array.isArray(value) ? value.map(String).filter(Boolean) : [];
            return [id, multiple ? (items.length > 0 ? items : undefined) : items[0]];
          }),
        );
        return { ...previous, ...values, page: 1 };
      });
    },
    [columnFilters, filterKeys, updateSearch],
  );

  const onQueryChange = React.useCallback(
    (q: string | undefined) => {
      updateSearch((previous) => ({ ...previous, q, page: 1 }));
    },
    [updateSearch],
  );

  const clearSearchKeys = React.useCallback(
    (keys: readonly Extract<keyof Search, string>[]) => {
      updateSearch((previous) => ({
        ...previous,
        ...Object.fromEntries(keys.map((key) => [key, undefined])),
        page: 1,
      }));
    },
    [updateSearch],
  );

  const isFiltered =
    search.q !== undefined ||
    Object.values(filters).some((value) => Array.isArray(value) && value.length > 0) ||
    scopeKeys.some((key) => search[key] !== undefined);

  return {
    q: search.q,
    page: search.page,
    per_page: search.per_page,
    sort: search.sort,
    filters,
    isFiltered,
    pagination,
    sorting,
    columnFilters,
    onPaginationChange,
    onSortingChange,
    onColumnFiltersChange,
    onQueryChange,
    clearSearchKeys,
  };
}

function singleSort(sorting: SortingState): SortingState {
  return sorting.length > 0 ? [sorting[0]] : [];
}
