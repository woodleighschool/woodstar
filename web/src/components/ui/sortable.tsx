import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import type { UniqueIdentifier } from "@dnd-kit/abstract";
import { arrayMove } from "@dnd-kit/helpers";
import { DragDropProvider } from "@dnd-kit/react";
import { isSortable, useSortable } from "@dnd-kit/react/sortable";
import * as React from "react";

import { useComposedRefs } from "@lib/compose-refs";
import { cn } from "@lib/utils";

interface SortableContextValue {
  itemIDs: UniqueIdentifier[];
}

const SortableRootContext = React.createContext<SortableContextValue | null>(null);

export interface SortableProps<TItem> {
  value: TItem[];
  onValueChange: (value: TItem[]) => void;
  getItemValue: (item: TItem) => UniqueIdentifier;
  children: React.ReactNode;
}

export function Sortable<TItem>({
  value,
  onValueChange,
  getItemValue,
  children,
}: SortableProps<TItem>) {
  const itemIDs = React.useMemo(() => value.map(getItemValue), [getItemValue, value]);
  const context = React.useMemo(() => ({ itemIDs }), [itemIDs]);

  return (
    <SortableRootContext.Provider value={context}>
      <DragDropProvider
        onDragEnd={(event) => {
          if (event.canceled) return;

          const { source } = event.operation;
          if (!isSortable(source) || source.initialIndex === source.index) return;

          onValueChange(arrayMove(value, source.initialIndex, source.index));
        }}
      >
        {children}
      </DragDropProvider>
    </SortableRootContext.Provider>
  );
}

export function SortableContent({ className, ...props }: React.ComponentProps<"div">) {
  const context = React.useContext(SortableRootContext);
  if (!context) throw new Error("SortableContent must be used inside Sortable");

  return <div data-slot="sortable-content" className={cn(className)} {...props} />;
}

interface SortableItemContextValue {
  handleRef: ReturnType<typeof useSortable>["handleRef"];
  disabled: boolean;
  dragging: boolean;
}

const SortableItemContext = React.createContext<SortableItemContextValue | null>(null);

export function SortableItem({
  value,
  disabled = false,
  className,
  ref,
  ...props
}: React.ComponentProps<"div"> & {
  value: UniqueIdentifier;
  disabled?: boolean;
}) {
  const root = React.useContext(SortableRootContext);
  if (!root) throw new Error("SortableItem must be used inside Sortable");

  const index = root.itemIDs.indexOf(value);
  const sortable = useSortable({ id: value, index, disabled });
  const composedRef = useComposedRefs(ref, sortable.ref);
  const context = React.useMemo<SortableItemContextValue>(
    () => ({
      handleRef: sortable.handleRef,
      disabled,
      dragging: sortable.isDragging,
    }),
    [disabled, sortable.handleRef, sortable.isDragging],
  );

  return (
    <SortableItemContext.Provider value={context}>
      <div
        ref={composedRef}
        data-slot="sortable-item"
        data-dragging={sortable.isDragging || undefined}
        className={cn(
          "data-dragging:relative data-dragging:z-10 data-dragging:opacity-80",
          className,
        )}
        {...props}
      />
    </SortableItemContext.Provider>
  );
}

interface SortableItemHandleProps
  extends React.ComponentProps<"button">, useRender.ComponentProps<"button"> {}

export function SortableItemHandle({
  disabled,
  className,
  ref,
  render,
  ...props
}: SortableItemHandleProps) {
  const context = React.useContext(SortableItemContext);
  if (!context) throw new Error("SortableItemHandle must be used inside SortableItem");

  const isDisabled = disabled ?? context.disabled;
  const composedRef = useComposedRefs(ref, context.handleRef);

  return useRender({
    defaultTagName: "button",
    props: mergeProps<"button">(
      {
        type: "button",
        "data-slot": "sortable-item-handle",
        "data-dragging": context.dragging || undefined,
        ref: composedRef,
        disabled: isDisabled,
        className: cn(
          "cursor-grab touch-none select-none disabled:pointer-events-none disabled:opacity-50 data-dragging:cursor-grabbing",
          className,
        ),
      } as React.ComponentProps<"button">,
      props,
    ),
    render,
    state: {
      disabled: isDisabled,
      dragging: context.dragging,
      slot: "sortable-item-handle",
    },
  });
}
