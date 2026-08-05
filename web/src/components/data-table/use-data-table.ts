import {
  type ExpandedState,
  type ColumnVisibilityState,
  type RowSelectionState,
  type SortingState,
  useTable,
} from "@tanstack/react-table";
import * as React from "react";

import { DATA_TABLE_DEFAULT_COLUMN } from "@components/data-table/data-table-sizing";
import {
  dataTableFeatures,
  type DataTableOptions,
  type DataTableRowData,
  type DataTableState,
} from "@components/data-table/types";
import type { DataTableQuery } from "@components/data-table/use-data-table-search";

interface UseDataTableProps<TData extends DataTableRowData>
  extends
    Omit<
      DataTableOptions<TData>,
      | "state"
      | "features"
      | "pageCount"
      | "rowCount"
      | "manualFiltering"
      | "manualPagination"
      | "manualSorting"
    >,
    Required<Pick<DataTableOptions<TData>, "pageCount" | "rowCount">> {
  initialState?: Omit<Partial<DataTableState>, "sorting"> & {
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

export function useDataTable<TData extends DataTableRowData>(props: UseDataTableProps<TData>) {
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
  const [columnVisibility, setColumnVisibility] = React.useState<ColumnVisibilityState>(
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

  return useTable({
    features: dataTableFeatures,
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
    manualPagination: true,
    manualSorting: true,
    manualFiltering: true,
    meta: tableProps.meta,
  });
}
