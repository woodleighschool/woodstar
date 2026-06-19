import type {
  ColumnFiltersState,
  PaginationState,
  SortingState,
  Updater,
} from "@tanstack/react-table";
import { parseAsArrayOf, parseAsInteger, parseAsString, useQueryStates } from "nuqs";
import * as React from "react";

import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { getSortingStateParser, serializeSortingState } from "@/lib/parsers";

// Encodes a single-column sort in the backend wire format.
export function encodeSort(id: string, desc = false): string {
  return serializeSortingState([{ id, desc }]);
}

export interface DataTableFilterKey {
  id: string;
}

const emptyFilterKeys: readonly DataTableFilterKey[] = [];

export interface DataTableQuery {
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
  pagination: PaginationState;
  sorting: SortingState;
  columnFilters: ColumnFiltersState;
  onPaginationChange: (updaterOrValue: Updater<PaginationState>) => void;
  onSortingChange: (updaterOrValue: Updater<SortingState>) => void;
  onColumnFiltersChange: (updaterOrValue: Updater<ColumnFiltersState>) => void;
}

const baseParsers = {
  q: parseAsString,
  page: parseAsInteger.withDefault(1),
  per_page: parseAsInteger.withDefault(DEFAULT_PAGE_SIZE),
  sort: getSortingStateParser().withDefault([]),
};

export function useDataTableSearch(
  filterKeys: readonly DataTableFilterKey[] = emptyFilterKeys,
): DataTableQuery {
  const [base, setBase] = useQueryStates(baseParsers);

  const filterParsers = React.useMemo(
    () => Object.fromEntries(filterKeys.map((key) => [key.id, parseAsArrayOf(parseAsString)])),
    [filterKeys],
  );
  const [raw, setFilters] = useQueryStates(filterParsers);

  const filters = React.useMemo(() => {
    const out: Record<string, string[] | undefined> = {};
    for (const [key, value] of Object.entries(raw)) {
      if (value == null || value.length === 0) continue;
      out[key] = value;
    }
    return out;
  }, [raw]);

  const pagination = React.useMemo<PaginationState>(
    () => ({
      pageIndex: base.page - 1,
      pageSize: base.per_page,
    }),
    [base.page, base.per_page],
  );

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
      void setBase({ page: next.pageIndex + 1, per_page: next.pageSize });
    },
    [pagination, setBase],
  );

  const onSortingChange = React.useCallback(
    (updaterOrValue: Updater<SortingState>) => {
      const next =
        typeof updaterOrValue === "function" ? updaterOrValue(base.sort) : updaterOrValue;
      void setBase({ sort: singleSort(next) });
    },
    [base.sort, setBase],
  );

  const onColumnFiltersChange = React.useCallback(
    (updaterOrValue: Updater<ColumnFiltersState>) => {
      const next =
        typeof updaterOrValue === "function" ? updaterOrValue(columnFilters) : updaterOrValue;
      const values = Object.fromEntries(
        filterKeys.map(({ id }) => {
          const value = next.find((filter) => filter.id === id)?.value;
          const items = Array.isArray(value) ? value.map(String).filter(Boolean) : [];
          return [id, items.length > 0 ? items : null];
        }),
      );
      void setBase({ page: null });
      void setFilters(values);
    },
    [columnFilters, filterKeys, setBase, setFilters],
  );

  return {
    q: base.q ?? undefined,
    page: base.page,
    per_page: base.per_page,
    sort: base.sort.length > 0 ? serializeSortingState(base.sort) : undefined,
    filters,
    pagination,
    sorting: base.sort,
    columnFilters,
    onPaginationChange,
    onSortingChange,
    onColumnFiltersChange,
  };
}

function singleSort(sorting: SortingState): SortingState {
  return sorting.length > 0 ? [sorting[0]] : [];
}
