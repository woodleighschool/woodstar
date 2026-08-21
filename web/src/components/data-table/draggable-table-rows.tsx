import type { UniqueIdentifier } from "@dnd-kit/abstract";
import { RestrictToVerticalAxis } from "@dnd-kit/abstract/modifiers";
import { arrayMove } from "@dnd-kit/helpers";
import { DragDropProvider } from "@dnd-kit/react";
import { isSortable, useSortable } from "@dnd-kit/react/sortable";
import * as React from "react";

import { Button } from "@components/ui/button";
import { TableRow } from "@components/ui/table";
import { useComposedRefs } from "@lib/compose-refs";
import { cn } from "@lib/utils";

interface DraggableTableRowsProps<TItem> {
  value: TItem[];
  onValueChange: (value: TItem[]) => void;
  getRowId: (item: TItem) => UniqueIdentifier;
  children: React.ReactNode;
}

interface DraggableTableRowsContextValue {
  rowIDs: UniqueIdentifier[];
}

const DraggableTableRowsContext = React.createContext<DraggableTableRowsContextValue | null>(null);

export function DraggableTableRows<TItem>({
  value,
  onValueChange,
  getRowId,
  children,
}: DraggableTableRowsProps<TItem>) {
  const rowIDs = React.useMemo(() => value.map(getRowId), [getRowId, value]);
  const context = React.useMemo(() => ({ rowIDs }), [rowIDs]);

  return (
    <DraggableTableRowsContext.Provider value={context}>
      <DragDropProvider
        modifiers={[RestrictToVerticalAxis]}
        onDragEnd={(event) => {
          if (event.canceled) return;

          const { source } = event.operation;
          if (!isSortable(source) || source.initialIndex === source.index) return;

          onValueChange(arrayMove(value, source.initialIndex, source.index));
        }}
      >
        {children}
      </DragDropProvider>
    </DraggableTableRowsContext.Provider>
  );
}

interface DraggableTableRowContextValue {
  handleRef: ReturnType<typeof useSortable>["handleRef"];
  disabled: boolean;
  dragging: boolean;
}

const DraggableTableRowContext = React.createContext<DraggableTableRowContextValue | null>(null);

export function DraggableTableRow({
  id,
  disabled = false,
  className,
  ref,
  ...props
}: Omit<React.ComponentProps<typeof TableRow>, "id"> & {
  id: UniqueIdentifier;
  disabled?: boolean;
}) {
  const rows = React.useContext(DraggableTableRowsContext);
  if (!rows) throw new Error("DraggableTableRow must be used inside DraggableTableRows");

  const index = rows.rowIDs.indexOf(id);
  const sortable = useSortable({ id, index, disabled });
  const composedRef = useComposedRefs(ref, sortable.ref);
  const context = React.useMemo<DraggableTableRowContextValue>(
    () => ({
      handleRef: sortable.handleRef,
      disabled,
      dragging: sortable.isDragging,
    }),
    [disabled, sortable.handleRef, sortable.isDragging],
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
  const composedRef = useComposedRefs(ref, context.handleRef);

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      {...props}
      ref={composedRef}
      disabled={isDisabled}
      data-dragging={context.dragging || undefined}
      className={cn(
        `
          cursor-grab touch-none
          data-dragging:cursor-grabbing
        `,
        className,
      )}
    />
  );
}
