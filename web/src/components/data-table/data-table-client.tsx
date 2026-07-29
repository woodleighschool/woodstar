import {
  type ColumnDef,
  type ColumnFiltersState,
  type ExpandedState,
  type FilterFn,
  getCoreRowModel,
  getExpandedRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type Row,
  type SortingState,
  type Table,
  useReactTable,
} from "@tanstack/react-table";
import type { ReactNode } from "react";
import { useState } from "react";

import { DataTable } from "@components/data-table/data-table";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { Input } from "@components/ui/input";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";

interface DataTableClientProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];
  empty?: ReactNode;
  exportOptions?: DataTableExportOptions<TData>;
  getRowCanExpand?: (row: Row<TData>) => boolean;
  getRowId?: (row: TData, index: number, parent?: Row<TData>) => string;
  getSearchText?: (row: TData) => string;
  initialSorting?: SortingState;
  renderSubRow?: (row: Row<TData>) => ReactNode;
  searchPlaceholder?: string;
  title?: ReactNode;
  toolbar?: (table: Table<TData>) => ReactNode;
}

export function DataTableClient<TData>({
  columns,
  data,
  empty,
  exportOptions,
  getRowCanExpand,
  getRowId,
  getSearchText,
  initialSorting = [],
  renderSubRow,
  searchPlaceholder,
  title,
  toolbar,
}: DataTableClientProps<TData>) {
  const [sorting, setSorting] = useState<SortingState>(initialSorting);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [expanded, setExpanded] = useState<ExpandedState>({});
  const [globalFilter, setGlobalFilter] = useState("");
  const globalFilterFn: FilterFn<TData> | undefined = getSearchText
    ? (row, _columnId, value) =>
        getSearchText(row.original).toLocaleLowerCase().includes(String(value).toLocaleLowerCase())
    : undefined;
  const table = useReactTable({
    columns,
    data,
    state: {
      sorting,
      columnFilters,
      expanded,
      globalFilter,
    },
    initialState: {
      pagination: {
        pageIndex: 0,
        pageSize: DEFAULT_PAGE_SIZE,
      },
    },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onExpandedChange: setExpanded,
    onGlobalFilterChange: setGlobalFilter,
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

  const showSearch = Boolean(searchPlaceholder);
  const controls =
    showSearch || toolbar ? (
      <div className="flex flex-wrap items-center gap-2">
        {showSearch ? (
          <Input
            value={globalFilter}
            onChange={(event) => setGlobalFilter(event.target.value)}
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
