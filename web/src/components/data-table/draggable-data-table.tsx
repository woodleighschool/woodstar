import { DndContext, PointerSensor, closestCenter, useSensor, useSensors, type DragEndEvent } from "@dnd-kit/core";
import { SortableContext, arrayMove, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { flexRender, getCoreRowModel, useReactTable, type ColumnDef } from "@tanstack/react-table";
import { GripVertical } from "lucide-react";
import { createContext, useContext, type CSSProperties, type ReactNode } from "react";

import { DataTableBodyRow, type DataTableBodyRowProps } from "@/components/data-table/data-table-row";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";

interface DraggableDataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  onRowReorder: (rows: TData[]) => void;
  getRowId?: (row: TData) => string;
  disabled?: boolean;
  isLoading?: boolean;
  empty: ReactNode;
  footer?: ReactNode;
  skeletonRows?: number;
}

const DEFAULT_GET_ROW_ID = (row: unknown): string => String((row as { id: string | number }).id);

export function DraggableDataTable<TData, TValue>({
  columns,
  data,
  onRowReorder,
  getRowId,
  disabled = false,
  isLoading = false,
  empty,
  footer,
  skeletonRows = 8,
}: DraggableDataTableProps<TData, TValue>) {
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }));
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getRowId: getRowId ?? DEFAULT_GET_ROW_ID,
  });
  const showSkeleton = isLoading && data.length === 0;
  const showEmpty = !showSkeleton && data.length === 0;
  const visibleRows = table.getRowModel().rows;
  const rowIds = visibleRows.map((row) => row.id);
  const skeletonRowIds = Array.from({ length: skeletonRows }, (_, row) => `skeleton-row-${row}`);
  const canReorderRows = !disabled && visibleRows.length > 1 && !showSkeleton && !showEmpty;

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = visibleRows.findIndex((row) => row.id === active.id);
    const newIndex = visibleRows.findIndex((row) => row.id === over.id);
    if (oldIndex < 0 || newIndex < 0) return;

    onRowReorder(
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
                <SortableDataTableRow key={row.id} row={row} />
              ))}
            </SortableContext>
          ) : (
            visibleRows.map((row) => <DataTableBodyRow key={row.id} row={row} />)
          )}
        </TableBody>
      </Table>
      {showEmpty ? <div className="flex min-h-72 justify-center px-4 py-12">{empty}</div> : null}
    </div>
  );

  return (
    <div className="flex min-w-0 flex-col gap-3">
      {canReorderRows ? (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          {tableContent}
        </DndContext>
      ) : (
        tableContent
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

export function DraggableDataTableRowDragHandle({ label = "Reorder row" }: { label?: string }) {
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

function SortableDataTableRow<TData>({ row }: DataTableBodyRowProps<TData>) {
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
      <DataTableBodyRow row={row} ref={setNodeRef} style={style} />
    </RowDragHandleContext.Provider>
  );
}
