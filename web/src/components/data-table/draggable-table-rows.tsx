import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  type UniqueIdentifier,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { restrictToVerticalAxis } from "@dnd-kit/modifiers";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import * as React from "react";

import { Button } from "@/components/ui/button";
import { TableRow } from "@/components/ui/table";
import { useComposedRefs } from "@/lib/compose-refs";
import { cn } from "@/lib/utils";

interface DraggableTableRowsProps<TItem> {
  value: TItem[];
  onValueChange: (value: TItem[]) => void;
  getRowId: (item: TItem) => UniqueIdentifier;
  children: React.ReactNode;
}

export function DraggableTableRows<TItem>({
  value,
  onValueChange,
  getRowId,
  children,
}: DraggableTableRowsProps<TItem>) {
  const id = React.useId();
  const rowIDs = React.useMemo(() => value.map(getRowId), [getRowId, value]);
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 4 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const activeIndex = rowIDs.indexOf(active.id);
    const overIndex = rowIDs.indexOf(over.id);
    if (activeIndex < 0 || overIndex < 0) return;

    onValueChange(arrayMove(value, activeIndex, overIndex));
  }

  return (
    <DndContext
      id={id}
      collisionDetection={closestCenter}
      modifiers={[restrictToVerticalAxis]}
      sensors={sensors}
      onDragEnd={handleDragEnd}
    >
      <SortableContext items={rowIDs} strategy={verticalListSortingStrategy}>
        {children}
      </SortableContext>
    </DndContext>
  );
}

interface DraggableTableRowContextValue {
  attributes: ReturnType<typeof useSortable>["attributes"];
  listeners: ReturnType<typeof useSortable>["listeners"];
  setActivatorNodeRef: ReturnType<typeof useSortable>["setActivatorNodeRef"];
  disabled: boolean;
  dragging: boolean;
}

const DraggableTableRowContext = React.createContext<DraggableTableRowContextValue | null>(null);

export function DraggableTableRow({
  id,
  disabled = false,
  className,
  style,
  ref,
  ...props
}: Omit<React.ComponentProps<typeof TableRow>, "id"> & {
  id: UniqueIdentifier;
  disabled?: boolean;
}) {
  const sortable = useSortable({ id, disabled });
  const composedRef = useComposedRefs(ref, sortable.setNodeRef);
  const context = React.useMemo<DraggableTableRowContextValue>(
    () => ({
      attributes: sortable.attributes,
      listeners: sortable.listeners,
      setActivatorNodeRef: sortable.setActivatorNodeRef,
      disabled,
      dragging: sortable.isDragging,
    }),
    [
      disabled,
      sortable.attributes,
      sortable.isDragging,
      sortable.listeners,
      sortable.setActivatorNodeRef,
    ],
  );

  return (
    <DraggableTableRowContext.Provider value={context}>
      <TableRow
        ref={composedRef}
        data-dragging={sortable.isDragging || undefined}
        className={cn(
          "data-dragging:relative data-dragging:z-10 data-dragging:opacity-80",
          className,
        )}
        style={{
          transform: CSS.Translate.toString(sortable.transform),
          transition: sortable.transition,
          ...style,
        }}
        {...props}
      />
    </DraggableTableRowContext.Provider>
  );
}

export function DraggableTableRowHandle({
  disabled,
  className,
  ref,
  ...props
}: React.ComponentProps<typeof Button>) {
  const context = React.useContext(DraggableTableRowContext);
  if (!context) {
    throw new Error("DraggableTableRowHandle must be used inside DraggableTableRow");
  }

  const isDisabled = disabled ?? context.disabled;
  const composedRef = useComposedRefs(ref, context.setActivatorNodeRef);

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-label="Drag to reorder"
      {...(isDisabled ? {} : context.attributes)}
      {...(isDisabled ? {} : context.listeners)}
      {...props}
      ref={composedRef}
      disabled={isDisabled}
      data-dragging={context.dragging || undefined}
      className={cn("cursor-grab touch-none data-dragging:cursor-grabbing", className)}
    />
  );
}
