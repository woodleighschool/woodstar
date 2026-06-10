import { Plus, Trash2 } from "lucide-react";
import { useRef } from "react";

import { EmptyPanel } from "@/components/empty-panel";
import { LabelPicker } from "@/components/labels/label-picker";
import { TargetSection } from "@/components/targeting/target-section";
import { Button } from "@/components/ui/button";
import type { LabelRef } from "@/lib/api";
import { normalizeLabelTargetSet, type LabelTargetSet } from "@/lib/targeting";
import { cn } from "@/lib/utils";

type LabelTargetSetRow = LabelRef;
export type LabelTargetSetDirection = keyof LabelTargetSet;

interface LabelTargetSetEditorProps {
  value: LabelTargetSet;
  onChange: (next: LabelTargetSet) => void;
  includeBuiltins?: boolean;
  targetTitle?: string;
  exclusionTitle?: string;
  targetButtonLabel?: string;
  exclusionButtonLabel?: string;
  targetPlaceholder?: string;
  exclusionPlaceholder?: string;
  className?: string;
}

export function LabelTargetSetEditor({
  value,
  onChange,
  includeBuiltins = true,
  targetTitle = "Include",
  exclusionTitle = "Exclude",
  targetButtonLabel = "Add Include",
  exclusionButtonLabel = "Add Exclude",
  targetPlaceholder = "Select Label",
  exclusionPlaceholder = "Select Label",
  className,
}: LabelTargetSetEditorProps) {
  const targetSet = normalizeLabelTargetSet(value);

  function add(direction: LabelTargetSetDirection) {
    onChange({
      ...targetSet,
      [direction]: [...targetSet[direction], { label_id: 0 }],
    });
  }

  function update(direction: LabelTargetSetDirection, index: number, labelID: number) {
    onChange({
      ...targetSet,
      [direction]: targetSet[direction].map((row, rowIndex) =>
        rowIndex === index ? { ...row, label_id: labelID } : row,
      ),
    });
  }

  function remove(direction: LabelTargetSetDirection, index: number) {
    onChange({
      ...targetSet,
      [direction]: targetSet[direction].filter((_, rowIndex) => rowIndex !== index),
    });
  }

  return (
    <div className={cn("flex flex-col gap-6", className)}>
      <LabelTargetSetColumn
        title={targetTitle}
        direction="include"
        rows={targetSet.include}
        includeBuiltins={includeBuiltins}
        buttonLabel={targetButtonLabel}
        placeholder={targetPlaceholder}
        onAdd={add}
        onUpdate={update}
        onRemove={remove}
      />
      <LabelTargetSetColumn
        title={exclusionTitle}
        direction="exclude"
        rows={targetSet.exclude}
        includeBuiltins={includeBuiltins}
        buttonLabel={exclusionButtonLabel}
        placeholder={exclusionPlaceholder}
        onAdd={add}
        onUpdate={update}
        onRemove={remove}
      />
    </div>
  );
}

interface LabelTargetSetColumnProps {
  title: string;
  direction: LabelTargetSetDirection;
  rows: LabelTargetSetRow[];
  includeBuiltins: boolean;
  buttonLabel: string;
  placeholder: string;
  onAdd: (direction: LabelTargetSetDirection) => void;
  onUpdate: (direction: LabelTargetSetDirection, index: number, labelID: number) => void;
  onRemove: (direction: LabelTargetSetDirection, index: number) => void;
}

function LabelTargetSetColumn({
  title,
  direction,
  rows,
  includeBuiltins,
  buttonLabel,
  placeholder,
  onAdd,
  onUpdate,
  onRemove,
}: LabelTargetSetColumnProps) {
  const keys = useRef(new WeakMap<LabelTargetSetRow, string>());
  const nextKey = useRef(0);

  function rowKey(row: LabelTargetSetRow) {
    const existing = keys.current.get(row);
    if (existing) return existing;
    const key = `${direction}-${nextKey.current}`;
    nextKey.current += 1;
    keys.current.set(row, key);
    return key;
  }

  return (
    <TargetSection
      title={title}
      action={
        <Button type="button" variant="outline" size="sm" onClick={() => onAdd(direction)}>
          <Plus data-icon="inline-start" />
          {buttonLabel}
        </Button>
      }
    >
      {rows.length === 0 ? (
        <EmptyPanel>{emptyStateMessage(direction)}</EmptyPanel>
      ) : (
        <div className="flex flex-col gap-2">
          {rows.map((row, index) => (
            <div key={rowKey(row)} className="flex min-w-0 items-center gap-2">
              <LabelPicker
                value={row.label_id > 0 ? [row.label_id] : []}
                onChange={(next) => onUpdate(direction, index, next[0] ?? 0)}
                selectionMode="single"
                includeBuiltins={includeBuiltins}
                placeholder={placeholder}
                required
                invalid={row.label_id <= 0}
              />
              <Button
                type="button"
                className="shrink-0"
                variant="ghost"
                size="icon"
                aria-label="Remove label"
                onClick={() => onRemove(direction, index)}
              >
                <Trash2 />
              </Button>
            </div>
          ))}
        </div>
      )}
    </TargetSection>
  );
}

function emptyStateMessage(direction: LabelTargetSetDirection) {
  return direction === "include" ? "No includes yet" : "No excludes yet";
}
