import { useState } from "react";

import { encodeSort } from "@components/data-table/use-data-table-search";
import { InputGroupLoadingAddon } from "@components/input-group-loading-addon";
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
  ComboboxValue,
  useComboboxAnchor,
} from "@components/ui/combobox";
import { Spinner } from "@components/ui/spinner";
import { useLabels } from "@features/labels/queries";
import type { Label } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";

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
  const selectedLabel = selected[0] ?? null;
  const anchorRef = useComboboxAnchor();

  if (labels.error) {
    return <p className="text-sm text-destructive">{labels.error.message}</p>;
  }

  if (selectionMode === "single") {
    return (
      <SingleLabelCombobox
        key={selectedLabel?.id ?? "none"}
        items={items}
        selected={selectedLabel}
        emptyPlaceholder={emptyPlaceholder}
        placeholder={placeholder}
        noLabelsMessage={noLabelsMessage}
        required={required}
        invalid={invalid}
        loading={labels.isLoading}
        onChange={onChange}
      />
    );
  }

  return (
    <Combobox
      multiple
      items={items}
      value={selected}
      itemToStringLabel={(label) => label.name}
      itemToStringValue={(label) => String(label.id)}
      isItemEqualToValue={(label, candidate) => label.id === candidate.id}
      onValueChange={(next) => onChange(next.map((label) => label.id))}
    >
      <ComboboxChips ref={anchorRef} className="h-auto min-h-9 pr-2">
        <ComboboxValue>
          {(current: Label[]) => (
            <>
              {current.map((label) => (
                <ComboboxChip key={label.id}>{label.name}</ComboboxChip>
              ))}
              <ComboboxChipsInput
                className="h-[calc(--spacing(5.5))] min-w-16 flex-1 p-0 text-sm"
                placeholder={
                  items.length === 0 ? (emptyPlaceholder ?? "No Labels Available") : placeholder
                }
                required={required && selected.length === 0}
                aria-invalid={invalid ? true : undefined}
              />
            </>
          )}
        </ComboboxValue>
        {labels.isLoading ? <Spinner className="size-3.5" /> : null}
      </ComboboxChips>
      {labels.isLoading ? null : (
        <ComboboxContent anchor={anchorRef}>
          <ComboboxEmpty>{items.length === 0 ? noLabelsMessage : "No Labels Found."}</ComboboxEmpty>
          <ComboboxList>{labelItem}</ComboboxList>
        </ComboboxContent>
      )}
    </Combobox>
  );
}

function SingleLabelCombobox({
  items,
  selected,
  emptyPlaceholder,
  placeholder,
  noLabelsMessage,
  required,
  invalid,
  loading,
  onChange,
}: {
  items: Label[];
  selected: Label | null;
  emptyPlaceholder?: string;
  placeholder: string;
  noLabelsMessage: string;
  required: boolean;
  invalid: boolean;
  loading: boolean;
  onChange: (value: number[]) => void;
}) {
  const [inputValue, setInputValue] = useState(selected?.name ?? "");

  return (
    <Combobox
      items={items}
      value={selected}
      inputValue={inputValue}
      itemToStringLabel={(label) => label.name}
      itemToStringValue={(label) => String(label.id)}
      isItemEqualToValue={(label, selectedLabel) => label.id === selectedLabel.id}
      onInputValueChange={setInputValue}
      onValueChange={(next) => {
        onChange(next ? [next.id] : []);
        setInputValue(next?.name ?? "");
      }}
    >
      <ComboboxInput
        className="w-full"
        placeholder={items.length === 0 ? (emptyPlaceholder ?? "No Labels Available") : placeholder}
        required={required}
        aria-invalid={invalid ? true : undefined}
        aria-busy={loading}
        showClear={inputValue !== ""}
        showTrigger={!loading}
      >
        {loading ? <InputGroupLoadingAddon /> : null}
      </ComboboxInput>
      {loading ? null : (
        <ComboboxContent>
          <ComboboxEmpty>{items.length === 0 ? noLabelsMessage : "No Labels Found."}</ComboboxEmpty>
          <ComboboxList>{labelItem}</ComboboxList>
        </ComboboxContent>
      )}
    </Combobox>
  );
}

function labelItem(label: Label) {
  return (
    <ComboboxItem key={label.id} value={label} className="gap-2">
      <span className="min-w-0 flex-1 truncate">{label.name}</span>
      <span className="text-muted-foreground tabular-nums">{label.hosts_count}</span>
    </ComboboxItem>
  );
}
