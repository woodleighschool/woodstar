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
}

const baseParsers = {
  q: parseAsString,
  page: parseAsInteger.withDefault(1),
  per_page: parseAsInteger.withDefault(DEFAULT_PAGE_SIZE),
  // Same parser useDataTable writes with, so the `sort` key has one owner.
  sort: getSortingStateParser().withDefault([]),
};

// Reads the nuqs-owned table state (the same URL keys useDataTable writes) so a
// list page can drive its data query. nuqs stays the single owner of these keys;
// this is purely a reader.
export function useDataTableSearch(filterKeys: readonly DataTableFilterKey[] = []): DataTableQuery {
  const [base] = useQueryStates(baseParsers);

  const filterParsers = React.useMemo(
    () => Object.fromEntries(filterKeys.map((key) => [key.id, parseAsArrayOf(parseAsString)])),
    [filterKeys],
  );
  const [raw] = useQueryStates(filterParsers);

  const filters: Record<string, string[] | undefined> = {};
  for (const [key, value] of Object.entries(raw)) {
    if (value == null || value.length === 0) continue;
    filters[key] = value;
  }

  return {
    q: base.q ?? undefined,
    page: base.page,
    per_page: base.per_page,
    sort: base.sort.length > 0 ? serializeSortingState(base.sort) : undefined,
    filters,
  };
}
