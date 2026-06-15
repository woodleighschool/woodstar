import {
  Combobox,
  ComboboxAnchor,
  ComboboxBadgeItem,
  ComboboxBadgeList,
  ComboboxCancel,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxTrigger,
} from "@/components/ui/combobox";
import { Skeleton } from "@/components/ui/skeleton";
import { encodeSort, MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import { useLabels, type Label as WoodstarLabel } from "@/hooks/use-labels";
import { useEffect, useState } from "react";

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
  const [singleInputValue, setSingleInputValue] = useState(selectedLabel?.name ?? "");

  useEffect(() => {
    setSingleInputValue(selectedLabel?.name ?? "");
  }, [selectedLabel?.id, selectedLabel?.name]);

  if (labels.isLoading) {
    return <Skeleton className="h-9 w-full" />;
  }
  if (labels.error) {
    return <p className="text-sm text-destructive">{labels.error.message}</p>;
  }

  if (selectionMode === "single") {
    return (
      <Combobox
        value={selectedLabel ? String(selectedLabel.id) : ""}
        inputValue={singleInputValue}
        onInputValueChange={setSingleInputValue}
        onValueChange={(next) => {
          const label = rows.find((candidate) => String(candidate.id) === next);
          onChange(label ? [label.id] : []);
          setSingleInputValue(label?.name ?? "");
        }}
      >
        <ComboboxAnchor className="w-full">
          <ComboboxInput
            placeholder={
              items.length === 0 ? (emptyPlaceholder ?? "No Labels Available") : placeholder
            }
            required={required}
            aria-invalid={invalid ? true : undefined}
          />
          {singleInputValue !== "" ? (
            <ComboboxCancel
              aria-label="Clear label"
              onClick={() => {
                setSingleInputValue("");
                onChange([]);
              }}
            />
          ) : null}
          <ComboboxTrigger aria-label="Open labels" />
        </ComboboxAnchor>
        <ComboboxContent>
          <ComboboxEmpty>{items.length === 0 ? noLabelsMessage : "No Labels Found."}</ComboboxEmpty>
          {items.map(labelItem)}
        </ComboboxContent>
      </Combobox>
    );
  }

  return (
    <Combobox
      multiple
      value={selectedValues}
      onValueChange={(next) => onChange(next.map((id) => Number(id)).filter(Number.isFinite))}
    >
      <ComboboxAnchor className="h-auto min-h-9 flex-wrap py-1.5 pr-2">
        <ComboboxBadgeList>
          {selected.map((label) => (
            <ComboboxBadgeItem key={label.id} value={String(label.id)}>
              {label.name}
            </ComboboxBadgeItem>
          ))}
        </ComboboxBadgeList>
        <ComboboxInput
          className="h-[calc(--spacing(5.5))] min-w-16 flex-1 px-0 py-0 text-sm"
          placeholder={
            items.length === 0 ? (emptyPlaceholder ?? "No Labels Available") : placeholder
          }
          required={required && selected.length === 0}
          aria-invalid={invalid ? true : undefined}
        />
        <ComboboxTrigger aria-label="Open labels" className="ml-auto" />
      </ComboboxAnchor>
      <ComboboxContent>
        <ComboboxEmpty>{items.length === 0 ? noLabelsMessage : "No Labels Found."}</ComboboxEmpty>
        {items.map(labelItem)}
      </ComboboxContent>
    </Combobox>
  );
}

function labelItem(label: WoodstarLabel) {
  return (
    <ComboboxItem key={label.id} value={String(label.id)} label={label.name} className="gap-2">
      <span className="min-w-0 flex-1 truncate">{label.name}</span>
      <span className="text-muted-foreground tabular-nums">{label.hosts_count}</span>
    </ComboboxItem>
  );
}
