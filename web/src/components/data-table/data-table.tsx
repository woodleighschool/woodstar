import { DndContext, PointerSensor, closestCenter, useSensor, useSensors, type DragEndEvent } from "@dnd-kit/core";
import { SortableContext, arrayMove, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type OnChangeFn,
  type PaginationState,
  type RowSelectionState,
  type SortingState,
  type Table as TanStackTable,
  type Updater,
  type VisibilityState,
} from "@tanstack/react-table";
import { GripVertical } from "lucide-react";
import { createContext, useContext, useMemo, useState, type CSSProperties, type ReactNode } from "react";

import { DataTablePagination } from "@/components/data-table/data-table-pagination";
import { DataTableBodyRow, type DataTableBodyRowProps } from "@/components/data-table/data-table-row";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  /** Total row count from the server (drives pagination). */
  totalCount: number;
  pagination: PaginationState;
  sorting: SortingState;
  onPaginationChange: OnChangeFn<PaginationState>;
  onSortingChange: OnChangeFn<SortingState>;

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
  toolbar?: ReactNode | ((table: TanStackTable<TData>) => ReactNode);
  /** Rendered when totalCount === 0. */
  empty: ReactNode;
  emptyClassName?: string;
  footer?: ReactNode;
  /** Skeleton row count during initial load. */
  skeletonRows?: number;
  perPageOptions?: readonly number[];
  onRowReorder?: (rows: TData[]) => void;
  rowReorderDisabled?: boolean;
  /**
   * Use TanStack's internal sorting instead of the server. Pass when the table
   * data is a fixed snapshot (e.g. query results) rather than a paginated list.
   */
  clientSort?: boolean;
}

const DEFAULT_GET_ROW_ID = (row: unknown): string => String((row as { id: string | number }).id);

export function DataTable<TData, TValue>({
  columns,
  data,
  totalCount,
  pagination,
  sorting,
  onPaginationChange,
  onSortingChange,
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
  emptyClassName,
  footer,
  skeletonRows = 8,
  perPageOptions,
  onRowReorder,
  rowReorderDisabled = false,
  clientSort = false,
}: DataTableProps<TData, TValue>) {
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }));
  const [localSorting, setLocalSorting] = useState<SortingState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

  const sortingState = clientSort ? localSorting : sorting;

  const rowSelection: RowSelectionState = useMemo(
    () => Object.fromEntries(selectedRowIds.map((id) => [id, true])),
    [selectedRowIds],
  );

  const handleSortingChange = (updater: Updater<SortingState>) => {
    const next = typeof updater === "function" ? updater(sortingState) : updater;
    setLocalSorting(next);
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
    rowCount: totalCount,
    state: { pagination, sorting: sortingState, columnVisibility, rowSelection },
    onPaginationChange,
    onSortingChange: clientSort ? handleSortingChange : onSortingChange,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: handleRowSelectionChange,
    getRowId: getRowId ?? DEFAULT_GET_ROW_ID,
  });

  const showSkeleton = isLoading && data.length === 0;
  const showEmpty = !showSkeleton && data.length === 0;
  const visibleRows = table.getRowModel().rows;
  const rowIds = visibleRows.map((row) => row.id);
  const skeletonRowIds = Array.from({ length: skeletonRows }, (_, row) => `skeleton-row-${row}`);
  const canReorderRows =
    onRowReorder !== undefined && !rowReorderDisabled && visibleRows.length > 1 && !showSkeleton && !showEmpty;

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = visibleRows.findIndex((row) => row.id === active.id);
    const newIndex = visibleRows.findIndex((row) => row.id === over.id);
    if (oldIndex < 0 || newIndex < 0) return;

    onRowReorder?.(
      arrayMove(
        visibleRows.map((row) => row.original),
        oldIndex,
        newIndex,
      ),
    );
  };

  const tableContent = (
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
          {showSkeleton ? (
            skeletonRowIds.map((rowId) => (
              <TableRow key={rowId}>
                {table.getVisibleLeafColumns().map((col) => (
                  <TableCell key={col.id}>
                    <Skeleton className="h-4 w-3/4" />
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : showEmpty ? null : canReorderRows ? (
            <SortableContext items={rowIds} strategy={verticalListSortingStrategy}>
              {visibleRows.map((row) => (
                <SortableDataTableRow
                  key={row.id}
                  row={row}
                  enableRowSelection={enableRowSelection}
                  rowHref={rowHref}
                  onRowClick={onRowClick}
                />
              ))}
            </SortableContext>
          ) : (
            visibleRows.map((row) => (
              <DataTableBodyRow
                key={row.id}
                row={row}
                enableRowSelection={enableRowSelection}
                rowHref={rowHref}
                onRowClick={onRowClick}
              />
            ))
          )}
        </TableBody>
      </Table>
      {showEmpty ? <div className={cn("flex min-h-72 justify-center px-4 py-12", emptyClassName)}>{empty}</div> : null}
    </div>
  );

  return (
    <div className="flex min-w-0 flex-col gap-3">
      {typeof toolbar === "function" ? toolbar(table) : toolbar}
      {enableRowSelection && selectedRowIds.length > 0 ? (
        <div className="bg-muted/50 flex min-h-10 items-center justify-between rounded-md border px-3 py-2">
          <div className="text-muted-foreground text-sm tabular-nums">{selectedRowIds.length} selected</div>
          <div className="flex items-center gap-2">{bulkActions}</div>
        </div>
      ) : null}
      {canReorderRows ? (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          {tableContent}
        </DndContext>
      ) : (
        tableContent
      )}
      {clientSort ? null : (
        <DataTablePagination table={table} totalCount={totalCount} perPageOptions={perPageOptions} />
      )}
      {footer}
    </div>
  );
}

type RowDragHandleContextValue = Pick<
  ReturnType<typeof useSortable>,
  "attributes" | "listeners" | "setActivatorNodeRef"
> | null;

const RowDragHandleContext = createContext<RowDragHandleContextValue>(null);

export function DataTableRowDragHandle({ label = "Reorder row" }: { label?: string }) {
  const drag = useContext(RowDragHandleContext);

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-sm"
      className={cn("cursor-grab active:cursor-grabbing", !drag && "cursor-default opacity-50")}
      aria-label={label}
      aria-disabled={!drag}
      disabled={!drag}
      ref={drag?.setActivatorNodeRef}
      {...drag?.attributes}
      {...drag?.listeners}
    >
      <GripVertical />
    </Button>
  );
}

function SortableDataTableRow<TData>({ row, enableRowSelection, rowHref, onRowClick }: DataTableBodyRowProps<TData>) {
  const { attributes, listeners, setActivatorNodeRef, setNodeRef, transform, transition, isDragging } = useSortable({
    id: row.id,
  });
  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.65 : 1,
    position: isDragging ? "relative" : undefined,
    zIndex: isDragging ? 1 : undefined,
  };

  return (
    <RowDragHandleContext.Provider value={{ attributes, listeners, setActivatorNodeRef }}>
      <DataTableBodyRow
        row={row}
        enableRowSelection={enableRowSelection}
        rowHref={rowHref}
        onRowClick={onRowClick}
        ref={setNodeRef}
        style={style}
      />
    </RowDragHandleContext.Provider>
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
