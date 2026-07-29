import {
  type ColumnDef,
  type ExpandedState,
  type FilterFn,
  getCoreRowModel,
  getExpandedRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type Row,
  type Table,
  useReactTable,
} from "@tanstack/react-table";
import * as React from "react";

import { DataTable } from "@components/data-table/data-table";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import type { DataTableQuery } from "@components/data-table/use-data-table-search";

interface DataTableClientProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];
  empty?: React.ReactNode;
  exportOptions?: DataTableExportOptions<TData>;
  getRowCanExpand?: (row: Row<TData>) => boolean;
  getRowId?: (row: TData, index: number, parent?: Row<TData>) => string;
  getSearchText?: (row: TData) => string;
  renderSubRow?: (row: Row<TData>) => React.ReactNode;
  searchPlaceholder?: string;
  tableState: DataTableQuery;
  title?: React.ReactNode;
  toolbar?: (table: Table<TData>) => React.ReactNode;
}

export function DataTableClient<TData>({
  columns,
  data,
  empty,
  exportOptions,
  getRowCanExpand,
  getRowId,
  getSearchText,
  renderSubRow,
  searchPlaceholder,
  tableState,
  title,
  toolbar,
}: DataTableClientProps<TData>) {
  const [expanded, setExpanded] = React.useState<ExpandedState>({});
  const { pagination, onPaginationChange } = tableState;
  const globalFilter = tableState.q ?? "";
  const globalFilterFn: FilterFn<TData> | undefined = getSearchText
    ? (row, _columnId, value) =>
        getSearchText(row.original).toLocaleLowerCase().includes(String(value).toLocaleLowerCase())
    : undefined;
  const table = useReactTable({
    columns,
    data,
    state: {
      sorting: tableState.sorting,
      columnFilters: tableState.columnFilters,
      pagination: tableState.pagination,
      expanded,
      globalFilter,
    },
    onSortingChange: tableState.onSortingChange,
    onColumnFiltersChange: tableState.onColumnFiltersChange,
    onPaginationChange,
    onExpandedChange: setExpanded,
    autoResetPageIndex: false,
    getRowCanExpand,
    getRowId,
    globalFilterFn,
    getCoreRowModel: getCoreRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    paginateExpandedRows: false,
  });
  const pageCount = Math.max(1, table.getPageCount());

  React.useEffect(() => {
    if (pagination.pageIndex < pageCount) return;

    onPaginationChange({
      ...pagination,
      pageIndex: pageCount - 1,
    });
  }, [onPaginationChange, pageCount, pagination]);

  const showSearch = Boolean(searchPlaceholder);
  const controls =
    showSearch || toolbar ? (
      <div className="flex flex-wrap items-center gap-2">
        {showSearch ? (
          <DataTableSearchInput
            value={tableState.q ?? ""}
            onValueChange={tableState.onQueryChange}
            placeholder={searchPlaceholder}
            className="h-8 w-full sm:w-64"
          />
        ) : null}
        {toolbar?.(table)}
      </div>
    ) : null;

  return (
    <DataTable
      table={table}
      empty={empty}
      exportOptions={exportOptions}
      heading={title}
      renderSubRow={renderSubRow}
    >
      {controls}
    </DataTable>
  );
}
