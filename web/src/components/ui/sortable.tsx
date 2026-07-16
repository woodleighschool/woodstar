import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  type DraggableAttributes,
  type DraggableSyntheticListeners,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  type UniqueIdentifier,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { restrictToParentElement } from "@dnd-kit/modifiers";
import {
  arrayMove,
  rectSortingStrategy,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import * as React from "react";

import { useComposedRefs } from "@/lib/compose-refs";
import { cn } from "@/lib/utils";

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
  const id = React.useId();
  const itemIDs = React.useMemo(() => value.map(getItemValue), [getItemValue, value]);
  const context = React.useMemo(() => ({ itemIDs }), [itemIDs]);
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 4 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const activeIndex = itemIDs.indexOf(active.id);
    const overIndex = itemIDs.indexOf(over.id);
    if (activeIndex < 0 || overIndex < 0) return;

    onValueChange(arrayMove(value, activeIndex, overIndex));
  }

  return (
    <SortableRootContext.Provider value={context}>
      <DndContext
        id={id}
        collisionDetection={closestCenter}
        modifiers={[restrictToParentElement]}
        sensors={sensors}
        onDragEnd={handleDragEnd}
        accessibility={{
          screenReaderInstructions: {
            draggable:
              "Press space to pick up an item. Use the arrow keys to move it, then press space to drop it. Press escape to cancel.",
          },
        }}
      >
        {children}
      </DndContext>
    </SortableRootContext.Provider>
  );
}

export function SortableContent({ className, ...props }: React.ComponentProps<"div">) {
  const context = React.useContext(SortableRootContext);
  if (!context) throw new Error("SortableContent must be used inside Sortable");

  return (
    <SortableContext items={context.itemIDs} strategy={rectSortingStrategy}>
      <div data-slot="sortable-content" className={cn(className)} {...props} />
    </SortableContext>
  );
}

interface SortableItemContextValue {
  attributes: DraggableAttributes;
  listeners: DraggableSyntheticListeners | undefined;
  setActivatorNodeRef: (node: HTMLElement | null) => void;
  disabled: boolean;
  dragging: boolean;
}

const SortableItemContext = React.createContext<SortableItemContextValue | null>(null);

export function SortableItem({
  value,
  disabled = false,
  className,
  style,
  ref,
  ...props
}: React.ComponentProps<"div"> & {
  value: UniqueIdentifier;
  disabled?: boolean;
}) {
  const sortable = useSortable({ id: value, disabled });
  const composedRef = useComposedRefs(ref, sortable.setNodeRef);
  const context = React.useMemo<SortableItemContextValue>(
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
    <SortableItemContext.Provider value={context}>
      <div
        ref={composedRef}
        data-slot="sortable-item"
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
  const composedRef = useComposedRefs(ref, context.setActivatorNodeRef);

  return useRender({
    defaultTagName: "button",
    props: mergeProps<"button">(
      {
        type: "button",
        "data-slot": "sortable-item-handle",
        "data-dragging": context.dragging || undefined,
        ...(isDisabled ? {} : context.attributes),
        ...(isDisabled ? {} : context.listeners),
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
