import { DndContext, PointerSensor, closestCenter, useSensor, useSensors, type DragEndEvent } from "@dnd-kit/core";
import { SortableContext, arrayMove, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVertical } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";

export interface SortableItem {
  id: number;
}

export function SortableList<TItem extends SortableItem>({
  items,
  onChange,
  renderItem,
}: {
  items: TItem[];
  onChange: (items: TItem[]) => void;
  renderItem: (item: TItem) => ReactNode;
}) {
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }));

  function handleDragEnd(event: DragEndEvent) {
    const activeID = Number(event.active.id);
    const overID = Number(event.over?.id);
    if (!overID || activeID === overID) return;
    const oldIndex = items.findIndex((item) => item.id === activeID);
    const newIndex = items.findIndex((item) => item.id === overID);
    if (oldIndex < 0 || newIndex < 0) return;
    onChange(arrayMove(items, oldIndex, newIndex));
  }

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
      <SortableContext items={items.map((item) => item.id)} strategy={verticalListSortingStrategy}>
        <div className="grid gap-2">
          {items.map((item) => (
            <SortableRow key={item.id} id={item.id}>
              {renderItem(item)}
            </SortableRow>
          ))}
        </div>
      </SortableContext>
    </DndContext>
  );
}

function SortableRow({ id, children }: { id: number; children: ReactNode }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });
  return (
    <div
      ref={setNodeRef}
      className="bg-card flex min-w-0 items-center gap-2 rounded-md border p-2"
      style={{ transform: CSS.Transform.toString(transform), transition, opacity: isDragging ? 0.65 : 1 }}
    >
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-8 shrink-0 cursor-grab active:cursor-grabbing"
        {...attributes}
        {...listeners}
      >
        <GripVertical className="size-4" />
      </Button>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}
