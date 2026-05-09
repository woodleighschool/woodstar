import { useNavigate, useSearch } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 50;

export interface TablePaginationState {
  page: number;
  perPage: number;
  orderKey?: string;
  orderDirection?: "asc" | "desc";
}

export interface TablePaginationSetters {
  setPage: (page: number) => void;
  setPerPage: (perPage: number) => void;
  setSort: (orderKey: string | undefined, orderDirection: "asc" | "desc" | undefined) => void;
  setFilter: (key: string, value: string | undefined) => void;
}

interface PaginationSearch {
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: "asc" | "desc";
  [key: string]: unknown;
}

/**
 * Reads page / per_page / order_key / order_direction from the URL and
 * returns typed state + setters that write back. Page 1 and the default
 * per-page are omitted from the URL so query strings stay tidy.
 */
export function useTablePaginationParams(defaults?: { perPage?: number }): {
  state: TablePaginationState;
  setters: TablePaginationSetters;
} {
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  const defaultPerPage = defaults?.perPage ?? DEFAULT_PER_PAGE;

  const state: TablePaginationState = useMemo(
    () => ({
      page: search.page ?? DEFAULT_PAGE,
      perPage: search.per_page ?? defaultPerPage,
      orderKey: search.order_key,
      orderDirection: search.order_direction,
    }),
    [search.page, search.per_page, search.order_key, search.order_direction, defaultPerPage],
  );

  const setPage = useCallback(
    (page: number) => {
      void navigate({
        search: ((prev: PaginationSearch) => ({ ...prev, page: page <= DEFAULT_PAGE ? undefined : page })) as never,
        replace: true,
      });
    },
    [navigate],
  );

  const setPerPage = useCallback(
    (perPage: number) => {
      void navigate({
        search: ((prev: PaginationSearch) => ({
          ...prev,
          per_page: perPage === defaultPerPage ? undefined : perPage,
          page: undefined,
        })) as never,
        replace: true,
      });
    },
    [navigate, defaultPerPage],
  );

  const setSort = useCallback(
    (orderKey: string | undefined, orderDirection: "asc" | "desc" | undefined) => {
      void navigate({
        search: ((prev: PaginationSearch) => ({
          ...prev,
          order_key: orderKey,
          order_direction: orderDirection,
          page: undefined,
        })) as never,
        replace: true,
      });
    },
    [navigate],
  );

  const setFilter = useCallback(
    (key: string, value: string | undefined) => {
      void navigate({
        search: ((prev: PaginationSearch) => {
          const next: PaginationSearch = { ...prev, page: undefined };
          if (value === undefined || value === "") delete next[key];
          else next[key] = value;
          return next;
        }) as never,
        replace: true,
      });
    },
    [navigate],
  );

  return { state, setters: { setPage, setPerPage, setSort, setFilter } };
}
