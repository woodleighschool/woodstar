import { Plus, Trash2 } from "lucide-react";
import { useRef } from "react";

import { LabelPicker } from "@/components/labels/label-picker";
import { Button } from "@/components/ui/button";
import { Field, FieldLabel } from "@/components/ui/field";
import type { TargetLabel } from "@/lib/api";
import { cn } from "@/lib/utils";

export type LabelScopeRow = TargetLabel;
export type LabelScopeEffect = LabelScopeRow["effect"];

interface LabelScopeEditorProps {
  value: LabelScopeRow[];
  onChange: (next: LabelScopeRow[]) => void;
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
  const includes = value.filter((row) => row.effect === "include");
  const excludes = value.filter((row) => row.effect === "exclude");

  function add(effect: LabelScopeEffect) {
    onChange([...value, { effect, label_id: 0 }]);
  }

  function update(effect: LabelScopeEffect, index: number, labelID: number) {
    let seen = -1;
    onChange(
      value.map((row) => {
        if (row.effect !== effect) return row;
        seen += 1;
        return seen === index ? { ...row, label_id: labelID } : row;
      }),
    );
  }

  function remove(effect: LabelScopeEffect, index: number) {
    let seen = -1;
    onChange(
      value.filter((row) => {
        if (row.effect !== effect) return true;
        seen += 1;
        return seen !== index;
      }),
    );
  }

  return (
    <div className={cn("grid gap-6 lg:grid-cols-2", className)}>
      <LabelScopeColumn
        title={targetTitle}
        effect="include"
        rows={includes}
        includeBuiltins={includeBuiltins}
        buttonLabel={targetButtonLabel}
        placeholder={targetPlaceholder}
        onAdd={add}
        onUpdate={update}
        onRemove={remove}
      />
      <LabelScopeColumn
        title={exclusionTitle}
        effect="exclude"
        rows={excludes}
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
  effect: LabelScopeEffect;
  rows: LabelScopeRow[];
  includeBuiltins: boolean;
  buttonLabel: string;
  placeholder: string;
  onAdd: (effect: LabelScopeEffect) => void;
  onUpdate: (effect: LabelScopeEffect, index: number, labelID: number) => void;
  onRemove: (effect: LabelScopeEffect, index: number) => void;
}

function LabelScopeColumn({
  title,
  effect,
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
    const key = `${effect}-${nextKey.current}`;
    nextKey.current += 1;
    keys.current.set(row, key);
    return key;
  }

  return (
    <Field className="gap-3">
      <div className="flex items-center justify-between gap-3">
        <FieldLabel>{title}</FieldLabel>
        <Button type="button" variant="outline" size="sm" onClick={() => onAdd(effect)}>
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
                onChange={(next) => onUpdate(effect, index, next[0] ?? 0)}
                selectionMode="single"
                includeBuiltins={includeBuiltins}
                placeholder={placeholder}
                required
                invalid={row.label_id <= 0}
              />
              <Button type="button" variant="ghost" size="icon" onClick={() => onRemove(effect, index)}>
                <Trash2 />
              </Button>
            </div>
          ))}
        </div>
      )}
    </Field>
  );
}
