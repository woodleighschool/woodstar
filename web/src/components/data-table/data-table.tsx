import { useNavigate } from "@tanstack/react-router";
import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type RowSelectionState,
  type SortingState,
  type Updater,
} from "@tanstack/react-table";
import { useMemo, useState, type ReactNode } from "react";

import { DataTablePagination } from "@/components/data-table/data-table-pagination";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";

const INTERACTIVE_SELECTOR =
  "a, button, input, label, select, textarea, [role=checkbox], [role=menuitem], [role=button], [role=dialog]";

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
  /** When set, the entire row navigates on click (skipping interactive children). */
  rowHref?: (row: TData) => { to: string; params?: Record<string, string> };
  /** When set, the entire row invokes an action on click (skipping interactive children). */
  onRowClick?: (row: TData) => void;
  /** Stable row id; defaults to (row as { id: string }).id. */
  getRowId?: (row: TData) => string;
  /** Slot above the table (toolbar with search + filters). */
  toolbar?: ReactNode;
  /** Rendered when totalCount === 0. */
  empty: ReactNode;
  /** Skeleton row count during initial load. */
  skeletonRows?: number;
  perPageOptions?: readonly number[];
  /**
   * Use TanStack's internal sorting instead of the server. Pass when the table
   * data is a fixed snapshot (e.g. query results) rather than a paginated list.
   * sort/onSortChange/page/onPageChange become no-ops in this mode.
   */
  clientSort?: boolean;
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
  onRowClick,
  getRowId,
  toolbar,
  empty,
  skeletonRows = 8,
  perPageOptions,
  clientSort = false,
}: DataTableProps<TData, TValue>) {
  const navigate = useNavigate();
  const [localSorting, setLocalSorting] = useState<SortingState>([]);

  const sortingState: SortingState = clientSort
    ? localSorting
    : sort.orderKey
      ? [{ id: sort.orderKey, desc: sort.orderDirection === "desc" }]
      : [];

  const rowSelection: RowSelectionState = useMemo(
    () => Object.fromEntries(selectedRowIds.map((id) => [id, true])),
    [selectedRowIds],
  );

  const handleSortingChange = (updater: Updater<SortingState>) => {
    const next = typeof updater === "function" ? updater(sortingState) : updater;
    if (clientSort) {
      setLocalSorting(next);
      return;
    }
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
    getSortedRowModel: clientSort ? getSortedRowModel() : undefined,
    manualPagination: !clientSort,
    manualSorting: !clientSort,
    enableRowSelection,
    state: { sorting: sortingState, rowSelection },
    onSortingChange: handleSortingChange,
    onRowSelectionChange: handleRowSelectionChange,
    getRowId: getRowId ?? DEFAULT_GET_ROW_ID,
  });

  const showSkeleton = isLoading && data.length === 0;
  const showEmpty = !showSkeleton && data.length === 0;
  const skeletonRowIds = Array.from({ length: skeletonRows }, (_, row) => `skeleton-row-${row}`);

  return (
    <div className="flex min-w-0 flex-col gap-3">
      {toolbar}
      {enableRowSelection && selectedRowIds.length > 0 ? (
        <div className="bg-muted/50 flex min-h-10 items-center justify-between rounded-md border px-3 py-2">
          <div className="text-muted-foreground text-sm tabular-nums">{selectedRowIds.length} selected</div>
          <div className="flex items-center gap-2">{bulkActions}</div>
        </div>
      ) : null}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((group) => (
              <TableRow key={group.id}>
                {group.headers.map((header) => (
                  <TableHead key={header.id} className={cn(header.column.columnDef.meta?.headClassName)}>
                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {showSkeleton
              ? skeletonRowIds.map((rowId) => (
                  <TableRow key={rowId}>
                    {table.getAllLeafColumns().map((col) => (
                      <TableCell key={col.id}>
                        <Skeleton className="h-4 w-3/4" />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              : showEmpty
                ? null
                : table.getRowModel().rows.map((row) => {
                    const linkProps = rowHref?.(row.original);
                    const hasRowAction = linkProps !== undefined || onRowClick !== undefined;
                    const cells = row.getVisibleCells();
                    const firstDataIndex = enableRowSelection ? 1 : 0;
                    return (
                      <TableRow
                        key={row.id}
                        className={cn(hasRowAction && "cursor-pointer")}
                        onClick={
                          hasRowAction
                            ? (e) => {
                                const target = e.target as HTMLElement;
                                if (target.closest(INTERACTIVE_SELECTOR)) return;
                                if (window.getSelection()?.toString()) return;
                                if (linkProps !== undefined) {
                                  void navigate({ to: linkProps.to, params: linkProps.params });
                                } else {
                                  onRowClick?.(row.original);
                                }
                              }
                            : undefined
                        }
                      >
                        {cells.map((cell, i) => (
                          <TableCell
                            key={cell.id}
                            className={cn(
                              cell.column.columnDef.meta?.cellClassName,
                              i === firstDataIndex && linkProps && "font-medium",
                            )}
                          >
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </TableCell>
                        ))}
                      </TableRow>
                    );
                  })}
          </TableBody>
        </Table>
        {showEmpty ? <div className="flex min-h-72 justify-center px-4 py-12">{empty}</div> : null}
      </div>
      {clientSort ? null : (
        <DataTablePagination
          page={page}
          perPage={perPage}
          totalCount={totalCount}
          visibleCount={data.length}
          onPageChange={onPageChange}
          onPerPageChange={onPerPageChange}
          perPageOptions={perPageOptions}
        />
      )}
    </div>
  );
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
      />
    ),
    cell: ({ row }) => (
      <Checkbox checked={row.getIsSelected()} onCheckedChange={(value) => row.toggleSelected(!!value)} />
    ),
    meta: {
      headClassName: "w-10",
      cellClassName: "w-10",
    },
  };
}
