import {
  type ExpandedState,
  getCoreRowModel,
  type RowSelectionState,
  type SortingState,
  type TableOptions,
  type TableState,
  useReactTable,
  type VisibilityState,
} from "@tanstack/react-table";
import * as React from "react";

import { DATA_TABLE_DEFAULT_COLUMN } from "@components/data-table/data-table-sizing";
import type { DataTableQuery } from "@components/data-table/use-data-table-search";

interface UseDataTableProps<TData>
  extends
    Omit<
      TableOptions<TData>,
      | "state"
      | "pageCount"
      | "rowCount"
      | "getCoreRowModel"
      | "manualFiltering"
      | "manualPagination"
      | "manualSorting"
    >,
    Required<Pick<TableOptions<TData>, "pageCount" | "rowCount">> {
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
  const {
    columns,
    pageCount,
    rowCount,
    initialState,
    tableState,
    enableRowSelection = false,
    ...tableProps
  } = props;

  const [rowSelection, setRowSelection] = React.useState<RowSelectionState>(
    initialState?.rowSelection ?? {},
  );
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>(
    initialState?.columnVisibility ?? {},
  );
  const [expanded, setExpanded] = React.useState<ExpandedState>(initialState?.expanded ?? {});
  const { pagination, onPaginationChange } = tableState;

  const normalizedPageCount = pageCount < 0 ? -1 : Math.max(1, pageCount);

  React.useEffect(() => {
    if (normalizedPageCount < 0) return;

    const lastPageIndex = normalizedPageCount - 1;
    if (pagination.pageIndex <= lastPageIndex) return;

    onPaginationChange({
      ...pagination,
      pageIndex: lastPageIndex,
    });
  }, [normalizedPageCount, onPaginationChange, pagination]);

  return useReactTable({
    ...tableProps,
    columns,
    initialState,
    pageCount: normalizedPageCount,
    rowCount,
    state: {
      pagination: tableState.pagination,
      sorting: tableState.sorting,
      columnVisibility,
      rowSelection,
      columnFilters: tableState.columnFilters,
      expanded,
    },
    defaultColumn: {
      ...DATA_TABLE_DEFAULT_COLUMN,
      ...tableProps.defaultColumn,
      enableColumnFilter: false,
    },
    enableColumnResizing: tableProps.enableColumnResizing ?? true,
    columnResizeMode: tableProps.columnResizeMode ?? "onChange",
    enableRowSelection,
    onRowSelectionChange: setRowSelection,
    onPaginationChange: tableState.onPaginationChange,
    onSortingChange: tableState.onSortingChange,
    onColumnFiltersChange: tableState.onColumnFiltersChange,
    onColumnVisibilityChange: setColumnVisibility,
    onExpandedChange: setExpanded,
    enableMultiSort: false,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    manualSorting: true,
    manualFiltering: true,
    meta: tableProps.meta,
  });
}
