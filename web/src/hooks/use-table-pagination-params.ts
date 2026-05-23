import { useNavigate, useSearch } from "@tanstack/react-router";
import type { OnChangeFn, PaginationState, SortingState } from "@tanstack/react-table";
import { useCallback, useMemo } from "react";

const DEFAULT_PAGE_INDEX = 0;
const DEFAULT_PAGE_SIZE = 50;

export interface TableSearchParams {
  q?: string;
  page_index?: number;
  page_size?: number;
  sort?: string;
  [key: string]: unknown;
}

export interface TableURLState {
  pagination: PaginationState;
  sorting: SortingState;
}

export interface TableURLSetters {
  setPagination: OnChangeFn<PaginationState>;
  setSorting: OnChangeFn<SortingState>;
  setFilter: (key: string, value: string | undefined) => void;
}

export function tableQueryParams(state: TableURLState): Pick<TableSearchParams, "page_index" | "page_size" | "sort"> {
  const sort = state.sorting.at(0);
  return {
    page_index: state.pagination.pageIndex,
    page_size: state.pagination.pageSize,
    sort: sort ? `${sort.id}.${sort.desc ? "desc" : "asc"}` : undefined,
  };
}

export function useTablePaginationParams(defaults?: { pageSize?: number }): {
  state: TableURLState;
  setters: TableURLSetters;
} {
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  const defaultPageSize = defaults?.pageSize ?? DEFAULT_PAGE_SIZE;

  const state: TableURLState = useMemo(
    () => ({
      pagination: {
        pageIndex: typeof search.page_index === "number" ? search.page_index : DEFAULT_PAGE_INDEX,
        pageSize: typeof search.page_size === "number" ? search.page_size : defaultPageSize,
      },
      sorting: parseSorting(search.sort),
    }),
    [defaultPageSize, search.page_index, search.page_size, search.sort],
  );

  const setPagination = useCallback<OnChangeFn<PaginationState>>(
    (updater) => {
      const next = typeof updater === "function" ? updater(state.pagination) : updater;
      void navigate({
        search: ((prev: TableSearchParams) => ({
          ...prev,
          page_index: next.pageIndex === DEFAULT_PAGE_INDEX ? undefined : next.pageIndex,
          page_size: next.pageSize === defaultPageSize ? undefined : next.pageSize,
        })) as never,
        replace: true,
      });
    },
    [defaultPageSize, navigate, state.pagination],
  );

  const setSorting = useCallback<OnChangeFn<SortingState>>(
    (updater) => {
      const next = typeof updater === "function" ? updater(state.sorting) : updater;
      const sort = next.at(0);
      void navigate({
        search: ((prev: TableSearchParams) => ({
          ...prev,
          sort: sort ? `${sort.id}.${sort.desc ? "desc" : "asc"}` : undefined,
          page_index: undefined,
        })) as never,
        replace: true,
      });
    },
    [navigate, state.sorting],
  );

  const setFilter = useCallback(
    (key: string, value: string | undefined) => {
      void navigate({
        search: ((prev: TableSearchParams) => {
          const next: TableSearchParams = { ...prev, page_index: undefined };
          if (value === undefined || value === "") delete next[key];
          else next[key] = value;
          return next;
        }) as never,
        replace: true,
      });
    },
    [navigate],
  );

  return { state, setters: { setPagination, setSorting, setFilter } };
}

function parseSorting(sort: unknown): SortingState {
  if (typeof sort !== "string" || sort === "") {
    return [];
  }
  const [id, direction] = sort.split(".");
  if (!id) {
    return [];
  }
  return [{ id, desc: direction === "desc" }];
}
