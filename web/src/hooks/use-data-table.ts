import {
  getCoreRowModel,
  type RowSelectionState,
  type SortingState,
  type TableOptions,
  type TableState,
  useReactTable,
  type VisibilityState,
} from "@tanstack/react-table";
import * as React from "react";

import type { DataTableQuery } from "@/hooks/use-data-table-search";

interface UseDataTableProps<TData>
  extends
    Omit<
      TableOptions<TData>,
      | "state"
      | "pageCount"
      | "getCoreRowModel"
      | "manualFiltering"
      | "manualPagination"
      | "manualSorting"
    >,
    Required<Pick<TableOptions<TData>, "pageCount">> {
  initialState?: Omit<Partial<TableState>, "sorting"> & {
    sorting?: SortingState;
  };
  tableState: Pick<
    DataTableQuery,
    | "pagination"
    | "sorting"
    | "columnFilters"
    | "onPaginationChange"
    | "onSortingChange"
    | "onColumnFiltersChange"
  >;
}

export function useDataTable<TData>(props: UseDataTableProps<TData>) {
  const { columns, pageCount = -1, initialState, tableState, ...tableProps } = props;

  const [rowSelection, setRowSelection] = React.useState<RowSelectionState>(
    initialState?.rowSelection ?? {},
  );
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>(
    initialState?.columnVisibility ?? {},
  );

  return useReactTable({
    ...tableProps,
    columns,
    initialState,
    pageCount,
    state: {
      pagination: tableState.pagination,
      sorting: tableState.sorting,
      columnVisibility,
      rowSelection,
      columnFilters: tableState.columnFilters,
    },
    defaultColumn: {
      ...tableProps.defaultColumn,
      enableColumnFilter: false,
    },
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    onPaginationChange: tableState.onPaginationChange,
    onSortingChange: tableState.onSortingChange,
    onColumnFiltersChange: tableState.onColumnFiltersChange,
    onColumnVisibilityChange: setColumnVisibility,
    enableMultiSort: false,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    manualSorting: true,
    manualFiltering: true,
    meta: tableProps.meta,
  });
}
