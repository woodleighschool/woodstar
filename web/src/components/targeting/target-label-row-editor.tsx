import { Plus, Trash2 } from "lucide-react";
import { useRef } from "react";

import { LabelPicker } from "@/components/labels/label-picker";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import type { TargetLabel } from "@/lib/api";

export type TargetLabelRow = TargetLabel;
export type TargetLabelEffect = TargetLabelRow["effect"];

interface TargetLabelRowEditorProps {
  value: TargetLabelRow[];
  onChange: (next: TargetLabelRow[]) => void;
  includeBuiltins?: boolean;
  noun: string;
}

export function TargetLabelRowEditor({ value, onChange, includeBuiltins = true, noun }: TargetLabelRowEditorProps) {
  const rows = value;
  const includes = rows.filter((row) => row.effect === "include");
  const excludes = rows.filter((row) => row.effect === "exclude");

  function add(effect: TargetLabelEffect) {
    onChange([...rows, { effect, label_id: 0 }]);
  }

  function update(effect: TargetLabelEffect, index: number, labelID: number) {
    let seen = -1;
    onChange(
      rows.map((row) => {
        if (row.effect !== effect) return row;
        seen += 1;
        return seen === index ? { ...row, label_id: labelID } : row;
      }),
    );
  }

  function remove(effect: TargetLabelEffect, index: number) {
    let seen = -1;
    onChange(
      rows.filter((row) => {
        if (row.effect !== effect) return true;
        seen += 1;
        return seen !== index;
      }),
    );
  }

  return (
    <div className="grid gap-4 rounded-md border p-4">
      <TargetLabelGroup
        title="Include"
        effect="include"
        rows={includes}
        noun={noun}
        includeBuiltins={includeBuiltins}
        onAdd={add}
        onUpdate={update}
        onRemove={remove}
      />
      <TargetLabelGroup
        title="Exclude"
        effect="exclude"
        rows={excludes}
        noun={noun}
        includeBuiltins={includeBuiltins}
        onAdd={add}
        onUpdate={update}
        onRemove={remove}
      />
    </div>
  );
}

interface TargetLabelGroupProps {
  title: "Include" | "Exclude";
  effect: TargetLabelEffect;
  rows: TargetLabelRow[];
  noun: string;
  includeBuiltins: boolean;
  onAdd: (effect: TargetLabelEffect) => void;
  onUpdate: (effect: TargetLabelEffect, index: number, labelID: number) => void;
  onRemove: (effect: TargetLabelEffect, index: number) => void;
}

function TargetLabelGroup({
  title,
  effect,
  rows,
  noun,
  includeBuiltins,
  onAdd,
  onUpdate,
  onRemove,
}: TargetLabelGroupProps) {
  const keys = useRef(new WeakMap<TargetLabelRow, string>());
  const nextKey = useRef(0);

  function rowKey(row: TargetLabelRow) {
    const existing = keys.current.get(row);
    if (existing) return existing;
    const key = `${effect}-${nextKey.current}`;
    nextKey.current += 1;
    keys.current.set(row, key);
    return key;
  }

  return (
    <div className="grid gap-2">
      <div className="flex items-center justify-between gap-3">
        <Label>{title}</Label>
        <Button type="button" variant="outline" size="sm" onClick={() => onAdd(effect)}>
          <Plus className="size-4" />
          {title}
        </Button>
      </div>
      {rows.length === 0 ? (
        <p className="text-muted-foreground rounded-md border border-dashed px-3 py-2 text-sm">No {effect} labels</p>
      ) : (
        <div className="grid gap-2">
          {rows.map((row, index) => (
            <div key={rowKey(row)} className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2">
              <LabelPicker
                value={row.label_id > 0 ? [row.label_id] : []}
                onChange={(next) => onUpdate(effect, index, next[0] ?? 0)}
                selectionMode="single"
                includeBuiltins={includeBuiltins}
                placeholder={`Select ${noun} label`}
                required
                invalid={row.label_id <= 0}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={`Remove ${effect} label`}
                onClick={() => onRemove(effect, index)}
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
