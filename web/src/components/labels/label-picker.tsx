import { useState } from "react";

import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
  ComboboxValue,
  useComboboxAnchor,
} from "@/components/ui/combobox";
import { Skeleton } from "@/components/ui/skeleton";
import { encodeSort } from "@/hooks/use-data-table-search";
import { useLabels } from "@/hooks/use-labels";
import type { Label as WoodstarLabel } from "@/lib/api";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

interface LabelPickerProps {
  value: number[];
  onChange: (value: number[]) => void;
  selectionMode?: "multiple" | "single";
  includeBuiltins?: boolean;
  unavailableLabelIDs?: readonly number[];
  emptyMessage?: string;
  emptyPlaceholder?: string;
  placeholder?: string;
  required?: boolean;
  invalid?: boolean;
}

export function LabelPicker({
  value,
  onChange,
  selectionMode = "multiple",
  includeBuiltins = false,
  unavailableLabelIDs = [],
  emptyMessage,
  emptyPlaceholder,
  placeholder = "Add Label",
  required = false,
  invalid = false,
}: LabelPickerProps) {
  const labels = useLabels({
    per_page: MAX_PAGE_SIZE,
    sort: encodeSort("name"),
    label_type: includeBuiltins ? undefined : "regular",
  });
  const rows = labels.data?.items ?? [];
  const unavailable = new Set(unavailableLabelIDs);
  const items = rows.filter(
    (label) =>
      (includeBuiltins || label.label_type === "regular") &&
      (value.includes(label.id) || !unavailable.has(label.id)),
  );
  const selected = rows.filter((label) => value.includes(label.id));
  const noLabelsMessage = emptyMessage ?? "No Labels Available.";
  const selectedValues = selected.map((label) => String(label.id));
  const selectedLabel = selected[0] ?? null;
  const anchorRef = useComboboxAnchor();

  if (labels.isLoading) {
    return <Skeleton className="h-9 w-full" />;
  }
  if (labels.error) {
    return <p className="text-sm text-destructive">{labels.error.message}</p>;
  }

  if (selectionMode === "single") {
    return (
      <SingleLabelCombobox
        key={selectedLabel?.id ?? "none"}
        rows={rows}
        items={items}
        selected={selectedLabel}
        emptyPlaceholder={emptyPlaceholder}
        placeholder={placeholder}
        noLabelsMessage={noLabelsMessage}
        required={required}
        invalid={invalid}
        onChange={onChange}
      />
    );
  }

  return (
    <Combobox
      multiple
      items={items.map((label) => String(label.id))}
      value={selectedValues}
      onValueChange={(next) => onChange(next.map((id) => Number(id)).filter(Number.isFinite))}
    >
      <ComboboxChips ref={anchorRef} className="h-auto min-h-9 pr-2">
        <ComboboxValue>
          {(current: string[]) =>
            current.map((id) => {
              const label = rows.find((candidate) => String(candidate.id) === id);
              return label ? <ComboboxChip key={label.id}>{label.name}</ComboboxChip> : null;
            })
          }
        </ComboboxValue>
        <ComboboxChipsInput
          className="h-[calc(--spacing(5.5))] min-w-16 flex-1 p-0 text-sm"
          placeholder={
            items.length === 0 ? (emptyPlaceholder ?? "No Labels Available") : placeholder
          }
          required={required && selected.length === 0}
          aria-invalid={invalid ? true : undefined}
        />
        <ComboboxTrigger className="ml-auto" />
      </ComboboxChips>
      <ComboboxContent anchor={anchorRef}>
        <ComboboxEmpty>{items.length === 0 ? noLabelsMessage : "No Labels Found."}</ComboboxEmpty>
        <ComboboxList>{items.map(labelItem)}</ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}

function SingleLabelCombobox({
  rows,
  items,
  selected,
  emptyPlaceholder,
  placeholder,
  noLabelsMessage,
  required,
  invalid,
  onChange,
}: {
  rows: WoodstarLabel[];
  items: WoodstarLabel[];
  selected: WoodstarLabel | null;
  emptyPlaceholder?: string;
  placeholder: string;
  noLabelsMessage: string;
  required: boolean;
  invalid: boolean;
  onChange: (value: number[]) => void;
}) {
  const [inputValue, setInputValue] = useState(selected?.name ?? "");

  return (
    <Combobox
      items={items.map((label) => String(label.id))}
      value={selected ? String(selected.id) : null}
      inputValue={inputValue}
      onInputValueChange={setInputValue}
      onValueChange={(next) => {
        const label = rows.find((candidate) => String(candidate.id) === next);
        onChange(label ? [label.id] : []);
        setInputValue(label?.name ?? "");
      }}
    >
      <ComboboxInput
        className="w-full"
        placeholder={items.length === 0 ? (emptyPlaceholder ?? "No Labels Available") : placeholder}
        required={required}
        aria-invalid={invalid ? true : undefined}
        showClear={inputValue !== ""}
      />
      <ComboboxContent>
        <ComboboxEmpty>{items.length === 0 ? noLabelsMessage : "No Labels Found."}</ComboboxEmpty>
        <ComboboxList>{items.map(labelItem)}</ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}

function labelItem(label: WoodstarLabel) {
  return (
    <ComboboxItem key={label.id} value={String(label.id)} className="gap-2">
      <span className="min-w-0 flex-1 truncate">{label.name}</span>
      <span className="text-muted-foreground tabular-nums">{label.hosts_count}</span>
    </ComboboxItem>
  );
}
