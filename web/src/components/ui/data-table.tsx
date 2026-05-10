import { Link } from "@tanstack/react-router";
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type RowSelectionState,
  type SortingState,
  type Updater,
} from "@tanstack/react-table";
import { useMemo, type ReactNode } from "react";

import { Checkbox } from "@/components/ui/checkbox";
import { DataTablePagination } from "@/components/ui/data-table-pagination";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";

export interface DataTableSort {
  orderKey?: string;
  orderDirection?: "asc" | "desc";
}

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  /** Total row count from the server (drives pagination). */
  totalCount: number;
  page: number;
  perPage: number;
  sort: DataTableSort;

  onPageChange: (page: number) => void;
  onPerPageChange: (perPage: number) => void;
  onSortChange: (next: DataTableSort) => void;

  isLoading?: boolean;
  enableRowSelection?: boolean;
  selectedRowIds?: string[];
  onSelectedRowIdsChange?: (ids: string[]) => void;
  bulkActions?: ReactNode;
  /** Optional row link wrapper. */
  rowHref?: (row: TData) => { to: string; params?: Record<string, string> };
  /** Stable row id; defaults to (row as { id: string }).id. */
  getRowId?: (row: TData) => string;
  /** Slot above the table (toolbar with search + filters). */
  toolbar?: ReactNode;
  /** Rendered when totalCount === 0. */
  empty: ReactNode;
  /** Skeleton row count during initial load. */
  skeletonRows?: number;
}

const DEFAULT_GET_ROW_ID = (row: unknown): string => String((row as { id: string | number }).id);

export function DataTable<TData, TValue>({
  columns,
  data,
  totalCount,
  page,
  perPage,
  sort,
  onPageChange,
  onPerPageChange,
  onSortChange,
  isLoading = false,
  enableRowSelection = false,
  selectedRowIds = [],
  onSelectedRowIdsChange,
  bulkActions,
  rowHref,
  getRowId,
  toolbar,
  empty,
  skeletonRows = 8,
}: DataTableProps<TData, TValue>) {
  const sortingState: SortingState = sort.orderKey ? [{ id: sort.orderKey, desc: sort.orderDirection === "desc" }] : [];
  const rowSelection: RowSelectionState = useMemo(
    () => Object.fromEntries(selectedRowIds.map((id) => [id, true])),
    [selectedRowIds],
  );

  const handleSortingChange = (updater: Updater<SortingState>) => {
    const next = typeof updater === "function" ? updater(sortingState) : updater;
    if (next.length === 0) onSortChange({ orderKey: undefined, orderDirection: undefined });
    else onSortChange({ orderKey: next[0].id, orderDirection: next[0].desc ? "desc" : "asc" });
  };

  const handleRowSelectionChange = (updater: Updater<RowSelectionState>) => {
    const next = typeof updater === "function" ? updater(rowSelection) : updater;
    onSelectedRowIdsChange?.(Object.keys(next).filter((id) => next[id]));
  };

  const tableColumns = useMemo(
    () => (enableRowSelection ? [selectionColumn<TData>(), ...columns] : columns),
    [columns, enableRowSelection],
  );

  // TanStack Table returns function-bearing objects; React Compiler cannot memoize this hook safely.

  const table = useReactTable({
    data,
    columns: tableColumns,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    manualSorting: true,
    enableRowSelection,
    state: { sorting: sortingState, rowSelection },
    onSortingChange: handleSortingChange,
    onRowSelectionChange: handleRowSelectionChange,
    getRowId: getRowId ?? DEFAULT_GET_ROW_ID,
  });

  const showSkeleton = isLoading && data.length === 0;

  return (
    <div className="flex flex-col gap-3">
      {toolbar}
      {enableRowSelection && selectedRowIds.length > 0 ? (
        <div className="bg-muted/50 flex min-h-10 items-center justify-between rounded-md border px-3 py-2">
          <div className="text-muted-foreground text-sm tabular-nums">
            {selectedRowIds.length} selected
          </div>
          <div className="flex items-center gap-2">{bulkActions}</div>
        </div>
      ) : null}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((group) => (
              <TableRow key={group.id}>
                {group.headers.map((header) => (
                  <TableHead key={header.id} className={cn((header.column.columnDef.meta as Meta)?.headClassName)}>
                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {showSkeleton ? (
              Array.from({ length: skeletonRows }).map((_, i) => (
                <TableRow key={`skeleton-${i}`}>
                  {table.getAllLeafColumns().map((col) => (
                    <TableCell key={col.id}>
                      <Skeleton className="h-4 w-3/4" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : data.length === 0 ? (
              <TableRow>
                <TableCell colSpan={tableColumns.length} className="p-0">
                  <div className="flex justify-center px-4 py-12">{empty}</div>
                </TableCell>
              </TableRow>
            ) : (
              table.getRowModel().rows.map((row) => {
                const linkProps = rowHref?.(row.original);
                const cells = row.getVisibleCells();
                if (linkProps) {
                  return (
                    <TableRow
                      key={row.id}
                      className="hover:bg-muted/50 cursor-pointer"
                      onClick={(e) => {
                        // ignore if user is selecting text or clicked a button/link
                        const target = e.target as HTMLElement;
                        if (target.closest("a, button, [role=menuitem]")) return;
                      }}
                    >
                      {cells.map((cell, i) => (
                        <TableCell
                          key={cell.id}
                          className={cn((cell.column.columnDef.meta as Meta)?.cellClassName, "p-0")}
                        >
                          <Link
                            to={linkProps.to}
                            params={linkProps.params}
                            className={cn("block px-4 py-2", i === 0 && "font-medium")}
                          >
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </Link>
                        </TableCell>
                      ))}
                    </TableRow>
                  );
                }
                return (
                  <TableRow key={row.id}>
                    {cells.map((cell) => (
                      <TableCell key={cell.id} className={cn((cell.column.columnDef.meta as Meta)?.cellClassName)}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>
      <DataTablePagination
        page={page}
        perPage={perPage}
        totalCount={totalCount}
        visibleCount={data.length}
        onPageChange={onPageChange}
        onPerPageChange={onPerPageChange}
      />
    </div>
  );
}

interface Meta {
  headClassName?: string;
  cellClassName?: string;
}

function selectionColumn<TData>(): ColumnDef<TData, unknown> {
  return {
    id: "__select",
    enableSorting: false,
    enableHiding: false,
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected() || (table.getIsSomePageRowsSelected() ? "indeterminate" : false)}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all rows"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
      />
    ),
    meta: {
      headClassName: "w-10",
      cellClassName: "w-10",
    },
  };
}
