import {
  type ColumnDef,
  type ColumnFiltersState,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type SortingState,
  type Table,
  useReactTable,
} from "@tanstack/react-table";
import type { ReactNode } from "react";
import { useState } from "react";

import { DataTable } from "@components/data-table/data-table";
import { Input } from "@components/ui/input";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";

interface DataTableClientProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];
  empty?: ReactNode;
  initialSorting?: SortingState;
  searchPlaceholder?: string;
  toolbar?: (table: Table<TData>) => ReactNode;
}

export function DataTableClient<TData>({
  columns,
  data,
  empty,
  initialSorting = [],
  searchPlaceholder,
  toolbar,
}: DataTableClientProps<TData>) {
  const [sorting, setSorting] = useState<SortingState>(initialSorting);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [globalFilter, setGlobalFilter] = useState("");
  const table = useReactTable({
    columns,
    data,
    state: {
      sorting,
      columnFilters,
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
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  });

  const controls =
    searchPlaceholder || toolbar ? (
      <div className="flex flex-wrap items-center gap-2">
        {searchPlaceholder ? (
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
    <DataTable table={table} empty={empty}>
      {controls}
    </DataTable>
  );
}
