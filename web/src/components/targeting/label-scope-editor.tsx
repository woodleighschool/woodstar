import { Plus, Trash2 } from "lucide-react";
import { useRef } from "react";

import { LabelPicker } from "@/components/labels/label-picker";
import { Button } from "@/components/ui/button";
import { Field, FieldLabel } from "@/components/ui/field";
import type { LabelRef } from "@/lib/api";
import { normalizeLabelTargetSet, type LabelTargetSet } from "@/lib/targeting";
import { cn } from "@/lib/utils";

export type LabelScopeRow = LabelRef;
export type LabelScopeDirection = keyof LabelTargetSet;

interface LabelScopeEditorProps {
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

export function LabelScopeEditor({
  value,
  onChange,
  includeBuiltins = true,
  targetTitle = "Targets",
  exclusionTitle = "Exclusions",
  targetButtonLabel = "Add Target",
  exclusionButtonLabel = "Add Exclusion",
  targetPlaceholder = "Select Label",
  exclusionPlaceholder = "Select Label",
  className,
}: LabelScopeEditorProps) {
  const targetSet = normalizeLabelTargetSet(value);

  function add(direction: LabelScopeDirection) {
    onChange({
      ...targetSet,
      [direction]: [...targetSet[direction], { label_id: 0 }],
    });
  }

  function update(direction: LabelScopeDirection, index: number, labelID: number) {
    onChange({
      ...targetSet,
      [direction]: targetSet[direction].map((row, rowIndex) =>
        rowIndex === index ? { ...row, label_id: labelID } : row,
      ),
    });
  }

  function remove(direction: LabelScopeDirection, index: number) {
    onChange({
      ...targetSet,
      [direction]: targetSet[direction].filter((_, rowIndex) => rowIndex !== index),
    });
  }

  return (
    <div className={cn("grid gap-6 lg:grid-cols-2", className)}>
      <LabelScopeColumn
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
      <LabelScopeColumn
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

interface LabelScopeColumnProps {
  title: string;
  direction: LabelScopeDirection;
  rows: LabelScopeRow[];
  includeBuiltins: boolean;
  buttonLabel: string;
  placeholder: string;
  onAdd: (direction: LabelScopeDirection) => void;
  onUpdate: (direction: LabelScopeDirection, index: number, labelID: number) => void;
  onRemove: (direction: LabelScopeDirection, index: number) => void;
}

function LabelScopeColumn({
  title,
  direction,
  rows,
  includeBuiltins,
  buttonLabel,
  placeholder,
  onAdd,
  onUpdate,
  onRemove,
}: LabelScopeColumnProps) {
  const keys = useRef(new WeakMap<LabelScopeRow, string>());
  const nextKey = useRef(0);

  function rowKey(row: LabelScopeRow) {
    const existing = keys.current.get(row);
    if (existing) return existing;
    const key = `${direction}-${nextKey.current}`;
    nextKey.current += 1;
    keys.current.set(row, key);
    return key;
  }

  return (
    <Field className="gap-3">
      <div className="flex items-center justify-between gap-3">
        <FieldLabel>{title}</FieldLabel>
        <Button type="button" variant="outline" size="sm" onClick={() => onAdd(direction)}>
          <Plus data-icon="inline-start" />
          {buttonLabel}
        </Button>
      </div>
      {rows.length === 0 ? (
        <div className="text-muted-foreground rounded-md border border-dashed px-3 py-2 text-sm">None</div>
      ) : (
        <div className="flex flex-col gap-2">
          {rows.map((row, index) => (
            <div key={rowKey(row)} className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2">
              <LabelPicker
                value={row.label_id > 0 ? [row.label_id] : []}
                onChange={(next) => onUpdate(direction, index, next[0] ?? 0)}
                selectionMode="single"
                includeBuiltins={includeBuiltins}
                placeholder={placeholder}
                required
                invalid={row.label_id <= 0}
              />
              <Button type="button" variant="ghost" size="icon" onClick={() => onRemove(direction, index)}>
                <Trash2 />
              </Button>
            </div>
          ))}
        </div>
      )}
    </Field>
  );
}
